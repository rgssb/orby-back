package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
)

var ErrReacaoIndisponivel = errors.New("Publicação ou comentário indisponível")

type Reacoes struct {
	db *sql.DB
}

func NovoRepositorioDeReacoes(db *sql.DB) *Reacoes {
	return &Reacoes{db: db}
}

func consultaDeReacoes(comentario bool) (tabela, chave, origem string) {
	if comentario {
		return "reacoes_comentarios", "comentario_id",
			"comentarios c join publicacoes p on p.id = c.publicacao_id join usuarios u on u.id = p.autor_id"
	}
	return "reacoes_publicacoes", "publicacao_id",
		"publicacoes p join usuarios u on u.id = p.autor_id"
}

func (repositorio Reacoes) Listar(id, usuarioID uint64, comentario bool) ([]modelos.Reacao, error) {
	tabela, chave, origem := consultaDeReacoes(comentario)
	referencia := "p.id"
	if comentario {
		referencia = "c.id"
	}
	var existe bool
	erro := repositorio.db.QueryRow("select exists(select 1 from "+origem+" where "+referencia+" = ? and "+postVisivel+")",
		id, usuarioID, usuarioID, usuarioID, usuarioID).Scan(&existe)
	if erro != nil {
		return nil, erro
	}
	if !existe {
		return nil, ErrReacaoIndisponivel
	}
	linhas, erro := repositorio.db.Query("select r.emoji, count(*), max(r.usuario_id = ?) from "+origem+
		" join "+tabela+" r on "+referencia+" = r."+chave+
		" where r."+chave+" = ? and "+postVisivel+" group by r.emoji order by r.emoji",
		usuarioID, id, usuarioID, usuarioID, usuarioID, usuarioID)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	reacoes := []modelos.Reacao{}
	for linhas.Next() {
		var reacao modelos.Reacao
		if erro = linhas.Scan(&reacao.Emoji, &reacao.Total, &reacao.Minha); erro != nil {
			return nil, erro
		}
		reacoes = append(reacoes, reacao)
	}
	return reacoes, linhas.Err()
}

func (repositorio Reacoes) Definir(id, usuarioID uint64, comentario bool, emoji string, ativa bool) error {
	tabela, chave, origem := consultaDeReacoes(comentario)
	referencia := "p.id"
	if comentario {
		referencia = "c.id"
	}
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	var publicacao uint64
	erro = tx.QueryRow("select p.id from "+origem+" where "+referencia+" = ? and "+postVisivel+" for update",
		id, usuarioID, usuarioID, usuarioID, usuarioID).Scan(&publicacao)
	if errors.Is(erro, sql.ErrNoRows) {
		return ErrReacaoIndisponivel
	}
	if erro != nil {
		return erro
	}
	if ativa {
		_, erro = tx.Exec("insert into "+tabela+" ("+chave+", usuario_id, emoji) values (?, ?, ?) on duplicate key update emoji = values(emoji)", id, usuarioID, emoji)
	} else {
		_, erro = tx.Exec("delete from "+tabela+" where "+chave+" = ? and usuario_id = ? and emoji = ?", id, usuarioID, emoji)
	}
	if erro != nil {
		return erro
	}
	return tx.Commit()
}
