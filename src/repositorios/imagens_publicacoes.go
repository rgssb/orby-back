package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/midia"
	"orby/api/src/modelos"
	"strings"
)

// MaximoDeImagens é a quantidade máxima de imagens de uma publicação
const MaximoDeImagens = 4

// ErrLimiteDeImagens é devolvido ao tentar anexar mais imagens que o permitido
var ErrLimiteDeImagens = errors.New("Uma publicação pode ter no máximo 4 imagens")

// A primeira imagem fica em imagens_publicacoes e as demais em imagens_extras_publicacoes (arquivos em
// midia.Publicacoes e midia.Extras). Nas rotas a imagem é indicada pela posição, começando em 0.

// ContarImagens devolve quantas imagens a publicação tem
func (repositorio Publicacoes) ContarImagens(publicacaoID uint64) (int, error) {
	var total int
	erro := repositorio.db.QueryRow(`
		select (select count(*) from imagens_publicacoes where publicacao_id = ?)
			+ (select count(*) from imagens_extras_publicacoes where publicacao_id = ?)`,
		publicacaoID, publicacaoID).Scan(&total)
	return total, erro
}

// AdicionarImagem anexa uma imagem na próxima posição livre
func (repositorio Publicacoes) AdicionarImagem(publicacaoID uint64, tipo string, dados []byte) error {
	return repositorio.AdicionarImagemComDescricao(publicacaoID, tipo, dados, "")
}

// AdicionarImagemComDescricao anexa uma imagem e seu texto alternativo.
func (repositorio Publicacoes) AdicionarImagemComDescricao(publicacaoID uint64, tipo string, dados []byte, descricao string) error {
	total, erro := repositorio.ContarImagens(publicacaoID)
	if erro != nil {
		return erro
	}
	if total >= MaximoDeImagens {
		return ErrLimiteDeImagens
	}

	if total == 0 {
		return repositorio.gravarPrincipal(publicacaoID, tipo, dados, descricao)
	}

	resultado, erro := repositorio.db.Exec(
		"insert into imagens_extras_publicacoes (publicacao_id, tipo, descricao) values (?, ?, ?)", publicacaoID, tipo, descricao)
	if erro != nil {
		return erro
	}
	id, erro := resultado.LastInsertId()
	if erro != nil {
		return erro
	}
	if erro = midia.Gravar(midia.Extras, uint64(id), dados); erro != nil {
		repositorio.db.Exec("delete from imagens_extras_publicacoes where id = ?", id)
		return erro
	}
	return nil
}

func (repositorio Publicacoes) gravarPrincipal(publicacaoID uint64, tipo string, dados []byte, descricao string) error {
	if erro := midia.Gravar(midia.Publicacoes, publicacaoID, dados); erro != nil {
		return erro
	}
	_, erro := repositorio.db.Exec(
		"replace into imagens_publicacoes (publicacao_id, tipo, descricao) values (?, ?, ?)", publicacaoID, tipo, descricao)
	return erro
}

// extraNaPosicao devolve o id e o tipo da imagem extra da posição (>= 1); sql.ErrNoRows quando não existe
func (repositorio Publicacoes) extraNaPosicao(publicacaoID uint64, posicao int) (uint64, string, error) {
	var id uint64
	var tipo string
	erro := repositorio.db.QueryRow(
		"select id, tipo from imagens_extras_publicacoes where publicacao_id = ? order by id limit 1 offset ?",
		publicacaoID, posicao-1).Scan(&id, &tipo)
	return id, tipo, erro
}

// AtualizarDescricaoImagem altera o texto alternativo sem modificar a imagem.
func (repositorio Publicacoes) AtualizarDescricaoImagem(publicacaoID uint64, posicao int, descricao string) error {
	if posicao < 0 || posicao >= MaximoDeImagens {
		return sql.ErrNoRows
	}
	if posicao == 0 {
		var id uint64
		if erro := repositorio.db.QueryRow("select publicacao_id from imagens_publicacoes where publicacao_id = ?", publicacaoID).Scan(&id); erro != nil {
			return erro
		}
		_, erro := repositorio.db.Exec("update imagens_publicacoes set descricao = ? where publicacao_id = ?", descricao, id)
		return erro
	}
	id, _, erro := repositorio.extraNaPosicao(publicacaoID, posicao)
	if erro != nil {
		return erro
	}
	_, erro = repositorio.db.Exec("update imagens_extras_publicacoes set descricao = ? where id = ?", descricao, id)
	return erro
}

// BuscarImagem devolve o tipo e os bytes da imagem na posição; sql.ErrNoRows quando não existe
func (repositorio Publicacoes) BuscarImagem(publicacaoID uint64, posicao int) (string, []byte, error) {
	if posicao < 0 || posicao >= MaximoDeImagens {
		return "", nil, sql.ErrNoRows
	}

	if posicao == 0 {
		var tipo string
		if erro := repositorio.db.QueryRow(
			"select tipo from imagens_publicacoes where publicacao_id = ?", publicacaoID,
		).Scan(&tipo); erro != nil {
			return "", nil, erro
		}
		dados, erro := lerMidia(midia.Publicacoes, publicacaoID)
		return tipo, dados, erro
	}

	id, tipo, erro := repositorio.extraNaPosicao(publicacaoID, posicao)
	if erro != nil {
		return "", nil, erro
	}
	dados, erro := lerMidia(midia.Extras, id)
	return tipo, dados, erro
}

// RemoverImagem apaga a imagem da posição; as seguintes sobem uma posição
func (repositorio Publicacoes) RemoverImagem(publicacaoID uint64, posicao int) error {
	if posicao < 0 || posicao >= MaximoDeImagens {
		return sql.ErrNoRows
	}

	if posicao > 0 {
		id, _, erro := repositorio.extraNaPosicao(publicacaoID, posicao)
		if erro != nil {
			return erro
		}
		if _, erro = repositorio.db.Exec("delete from imagens_extras_publicacoes where id = ?", id); erro != nil {
			return erro
		}
		return midia.Remover(midia.Extras, id)
	}

	resultado, erro := repositorio.db.Exec("delete from imagens_publicacoes where publicacao_id = ?", publicacaoID)
	if erro != nil {
		return erro
	}
	if linhas, _ := resultado.RowsAffected(); linhas == 0 {
		return sql.ErrNoRows
	}
	if erro = midia.Remover(midia.Publicacoes, publicacaoID); erro != nil {
		return erro
	}

	// a primeira extra vira a principal
	id, tipo, erro := repositorio.extraNaPosicao(publicacaoID, 1)
	if errors.Is(erro, sql.ErrNoRows) {
		return nil
	}
	if erro != nil {
		return erro
	}
	dados, erro := lerMidia(midia.Extras, id)
	if erro != nil {
		return erro
	}
	var descricao string
	if erro = repositorio.db.QueryRow("select descricao from imagens_extras_publicacoes where id = ?", id).Scan(&descricao); erro != nil {
		return erro
	}
	if erro = repositorio.gravarPrincipal(publicacaoID, tipo, dados, descricao); erro != nil {
		return erro
	}
	if _, erro = repositorio.db.Exec("delete from imagens_extras_publicacoes where id = ?", id); erro != nil {
		return erro
	}
	return midia.Remover(midia.Extras, id)
}

func (repositorio Publicacoes) idsDeImagensExtras(publicacaoID uint64) ([]uint64, error) {
	linhas, erro := repositorio.db.Query("select id from imagens_extras_publicacoes where publicacao_id = ?", publicacaoID)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	var ids []uint64
	for linhas.Next() {
		var id uint64
		if erro = linhas.Scan(&id); erro != nil {
			return nil, erro
		}
		ids = append(ids, id)
	}
	return ids, linhas.Err()
}

// removerArquivosDeImagens apaga os arquivos de uma publicação já excluída do banco
func (repositorio Publicacoes) removerArquivosDeImagens(publicacaoID uint64, extras []uint64) error {
	for _, id := range extras {
		if erro := midia.Remover(midia.Extras, id); erro != nil {
			return erro
		}
	}
	return midia.Remover(midia.Publicacoes, publicacaoID)
}

// anexarImagens preenche Imagens (quantidade) em cada publicação
func (repositorio Publicacoes) descricoesDeImagens(ids []interface{}) (map[uint64][]string, error) {
	descricoes := make(map[uint64][]string)
	if len(ids) == 0 {
		return descricoes, nil
	}
	marcadores := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	argumentos := append(append([]interface{}{}, ids...), ids...)
	linhas, erro := repositorio.db.Query(`
		select publicacao_id, descricao from (
			select publicacao_id, descricao, 0 as ordem, 0 as imagem_id from imagens_publicacoes where publicacao_id in (`+marcadores+`)
			union all
			select publicacao_id, descricao, 1 as ordem, id as imagem_id from imagens_extras_publicacoes where publicacao_id in (`+marcadores+`)
		) imagens order by publicacao_id, ordem, imagem_id`, argumentos...)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	for linhas.Next() {
		var id uint64
		var descricao string
		if erro = linhas.Scan(&id, &descricao); erro != nil {
			return nil, erro
		}
		descricoes[id] = append(descricoes[id], descricao)
	}
	return descricoes, linhas.Err()
}

func (repositorio Publicacoes) anexarImagens(publicacoes []modelos.Publicacao, ids []interface{}) error {
	descricoes, erro := repositorio.descricoesDeImagens(ids)
	if erro != nil {
		return erro
	}
	for i := range publicacoes {
		publicacoes[i].DescricoesImagens = descricoes[publicacoes[i].ID]
		publicacoes[i].Imagens = len(publicacoes[i].DescricoesImagens)
	}
	return nil
}
