package repositorios

import (
	"database/sql"
	"orby/api/src/modelos"
	"regexp"
	"strings"
)

// colunaTemImagem informa se a publicacao possui imagem anexada
// postVisivel filtra publicações de perfis privados: só o dono e quem o segue as veem (usa o viewer duas vezes)
const postVisivel = "(u.privado = 0 OR u.id = ? OR EXISTS (SELECT 1 FROM seguidores sx WHERE sx.usuario_id = u.id AND sx.seguidor_id = ?))"

const colunaTemImagem = "exists(select 1 from imagens_publicacoes ip where ip.publicacao_id = p.id)"

// Publicacoes representa um repositorio de publicacoes
type Publicacoes struct {
	db *sql.DB
}

// NovoRepositorioDePublicacoes cria um novo repositorio de publicacoes
func NovoRepositorioDePublicacoes(db *sql.DB) *Publicacoes {
	return &Publicacoes{db: db}
}

// Criar insere uma publicacao no banco de dados
func (repositorio *Publicacoes) Criar(publicacao modelos.Publicacao) (uint64, error) {
	statement, erro := repositorio.db.Prepare(
		"insert into publicacoes (titulo, conteudo, autor_id) VALUES (?, ?, ?)",
	)

	if erro != nil {
		return 0, erro
	}

	defer statement.Close()

	resultado, erro := statement.Exec(publicacao.Titulo, publicacao.Conteudo, publicacao.AutorID)
	if erro != nil {
		return 0, erro
	}

	ultimoID, erro := resultado.LastInsertId()
	if erro != nil {
		return 0, erro
	}

	return uint64(ultimoID), nil
}

// BuscarPorID busca uma publicacao por ID
func (repositorio Publicacoes) BuscarPorID(publicacaoID uint64) (modelos.Publicacao, error) {
	var publicacao modelos.Publicacao
	var editadaEm sql.NullTime
	erro := repositorio.db.QueryRow(`
		select p.*, u.nick, u.suspenso_em is not null, `+colunaFotoVersao+`,
			(select count(*) from comentarios c where c.publicacao_id = p.id),
			`+colunaTemImagem+`
		from publicacoes p inner join usuarios u on u.id = p.autor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where p.id = ?`,
		publicacaoID,
	).Scan(
		&publicacao.ID,
		&publicacao.Titulo,
		&publicacao.Conteudo,
		&publicacao.AutorID,
		&publicacao.Curtidas,
		&publicacao.CriadaEm,
		&editadaEm,
		&publicacao.AutorNick,
		&publicacao.AutorSuspenso,
		&publicacao.AutorFotoVersao,
		&publicacao.Comentarios,
		&publicacao.TemImagem,
	)
	if erro != nil {
		return modelos.Publicacao{}, erro
	}

	publicacao.Editado = editadaEm.Valid
	return publicacao, nil

}

// Busca traz as publucacoes dos usuarios seguidos e tambem do proprio usuario que fez a requisicao
// limite igual a zero traz todas; deslocamento pula as primeiras publicacoes (paginacao).
func (repositorio Publicacoes) Buscar(usuarioID, limite, deslocamento uint64, aba, ordem string) ([]modelos.Publicacao, error) {
	filtro := "(u.id = ? OR EXISTS (SELECT 1 FROM seguidores s WHERE s.usuario_id = p.autor_id AND s.seguidor_id = ?))"
	argumentos := []interface{}{usuarioID, usuarioID, usuarioID, usuarioID}
	if aba == "todos" {
		filtro = postVisivel
	}
	// quem foi silenciado some dos feeds, mas continua nos perfis e nas buscas
	filtro += " AND NOT EXISTS (SELECT 1 FROM silenciados sl WHERE sl.silenciador_id = ? AND sl.silenciado_id = p.autor_id)"
	argumentos = append(argumentos, usuarioID)
	// "Em alta": posts dos últimos 7 dias; curtidas e comentários pesam igual na pontuação
	ordenacao := "p.id DESC"
	if ordem == "populares" {
		filtro += " AND p.criadoEm >= NOW() - INTERVAL 7 DAY"
		ordenacao = "(p.curtidas + (SELECT COUNT(*) FROM comentarios c WHERE c.publicacao_id = p.id)) DESC, p.id DESC"
	}

	consulta := `
		SELECT p.*, u.nick, u.suspenso_em is not null, ` + colunaFotoVersao + `,
			(SELECT COUNT(*) FROM comentarios c WHERE c.publicacao_id = p.id),
			` + colunaTemImagem + `,
			EXISTS (
				SELECT 1 FROM curtidas_publicacoes cp
				WHERE cp.publicacao_id = p.id AND cp.usuario_id = ?
			) AS curtido_por_usuario,
			EXISTS (SELECT 1 FROM salvos sv WHERE sv.publicacao_id = p.id AND sv.usuario_id = ?) AS salvo_por_usuario
		FROM publicacoes p
		INNER JOIN usuarios u ON u.id = p.autor_id
		LEFT JOIN fotos_usuarios f ON f.usuario_id = u.id
		WHERE ` + filtro + `
		ORDER BY ` + ordenacao

	if limite > 0 {
		consulta += " LIMIT ? OFFSET ?"
		argumentos = append(argumentos, limite, deslocamento)
	}

	linhas, erro := repositorio.db.Query(consulta, argumentos...)

	if erro != nil {
		return nil, erro
	}

	defer linhas.Close()

	var publicacoes []modelos.Publicacao

	for linhas.Next() {
		var publicacao modelos.Publicacao
		var editadaEm sql.NullTime

		if erro = linhas.Scan(
			&publicacao.ID,
			&publicacao.Titulo,
			&publicacao.Conteudo,
			&publicacao.AutorID,
			&publicacao.Curtidas,
			&publicacao.CriadaEm,
			&editadaEm,
			&publicacao.AutorNick,
			&publicacao.AutorSuspenso,
			&publicacao.AutorFotoVersao,
			&publicacao.Comentarios,
			&publicacao.TemImagem,
			&publicacao.CurtidoPorUsuario,
			&publicacao.SalvoPorUsuario,
		); erro != nil {
			return nil, erro
		}

		publicacao.Editado = editadaEm.Valid
		publicacoes = append(publicacoes, publicacao)
	}

	if erro = linhas.Err(); erro != nil {
		return nil, erro
	}

	return publicacoes, nil
}

// CurtidaPor informa se o usuario curtiu a publicacao
func (repositorio Publicacoes) CurtidaPor(publicacaoID, usuarioID uint64) (bool, error) {
	var curtida bool
	erro := repositorio.db.QueryRow(
		"select exists (select 1 from curtidas_publicacoes where publicacao_id = ? and usuario_id = ?)",
		publicacaoID, usuarioID,
	).Scan(&curtida)
	return curtida, erro
}

// Atualizar altera a publicacao de um usuario
func (repositorio Publicacoes) Atualizar(publicacaoID uint64, publicacao modelos.Publicacao) error {
	statement, erro := repositorio.db.Prepare(
		"update publicacoes set titulo = ?, conteudo = ?, editadoEm = current_timestamp() where id = ?",
	)

	if erro != nil {
		return erro
	}

	defer statement.Close()

	if _, erro = statement.Exec(publicacao.Titulo, publicacao.Conteudo, publicacaoID); erro != nil {
		return erro
	}

	return nil
}

// Deletar exclui uma publicacao do banco de dados
func (repositorio Publicacoes) Deletar(publicacaoID uint64) error {
	extras, erro := repositorio.idsDeImagensExtras(publicacaoID)
	if erro != nil {
		return erro
	}

	// os reposts não fazem sentido sem o original
	if _, erro := repositorio.db.Exec("delete from publicacoes where id in (select publicacao_id from reposts where original_id = ?)", publicacaoID); erro != nil {
		return erro
	}

	statement, erro := repositorio.db.Prepare(
		"delete from publicacoes where id = ?",
	)

	if erro != nil {
		return erro
	}

	defer statement.Close()

	if _, erro = statement.Exec(publicacaoID); erro != nil {
		return erro
	}

	return repositorio.removerArquivosDeImagens(publicacaoID, extras)
}

// BuscarPorUsuario busca todas as publicacoes de um usuario no banco de dados
func (repositorio Publicacoes) BuscarPorUsuario(usuarioID, viewerID uint64) ([]modelos.Publicacao, error) {
	linhas, erro := repositorio.db.Query(`
		select p.*, u.nick, u.suspenso_em is not null, `+colunaFotoVersao+`,
			(select count(*) from comentarios c where c.publicacao_id = p.id),
			`+colunaTemImagem+`,
			EXISTS (
				SELECT 1 FROM curtidas_publicacoes cp
				WHERE cp.publicacao_id = p.id AND cp.usuario_id = ?
			) AS curtido_por_usuario,
			EXISTS (SELECT 1 FROM salvos sv WHERE sv.publicacao_id = p.id AND sv.usuario_id = ?) AS salvo_por_usuario
		from publicacoes p
		join usuarios u on u.id = p.autor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where p.autor_id = ?
		order by p.id desc`, viewerID, viewerID, usuarioID)

	if erro != nil {
		return nil, erro
	}

	defer linhas.Close()

	var publicacoes []modelos.Publicacao

	for linhas.Next() {
		var publicacao modelos.Publicacao
		var editadaEm sql.NullTime

		if erro = linhas.Scan(
			&publicacao.ID,
			&publicacao.Titulo,
			&publicacao.Conteudo,
			&publicacao.AutorID,
			&publicacao.Curtidas,
			&publicacao.CriadaEm,
			&editadaEm,
			&publicacao.AutorNick,
			&publicacao.AutorSuspenso,
			&publicacao.AutorFotoVersao,
			&publicacao.Comentarios,
			&publicacao.TemImagem,
			&publicacao.CurtidoPorUsuario,
			&publicacao.SalvoPorUsuario,
		); erro != nil {
			return nil, erro
		}

		publicacao.Editado = editadaEm.Valid
		publicacoes = append(publicacoes, publicacao)
	}

	if erro = linhas.Err(); erro != nil {
		return nil, erro
	}

	return publicacoes, nil
}

// Curtir registra uma curtida do usuario e atualiza o total quando ela é nova.
func (repositorio Publicacoes) Curtir(publicacaoID, usuarioID uint64) error {
	return repositorio.alterarCurtida(publicacaoID, usuarioID, true)
}

// Descurtir remove a curtida do usuario e atualiza o total quando ela existia.
func (repositorio Publicacoes) Descurtir(publicacaoID, usuarioID uint64) error {
	return repositorio.alterarCurtida(publicacaoID, usuarioID, false)
}

func (repositorio Publicacoes) alterarCurtida(publicacaoID, usuarioID uint64, curtir bool) error {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer transacao.Rollback()

	var id uint64
	if erro = transacao.QueryRow(
		"SELECT id FROM publicacoes WHERE id = ? FOR UPDATE",
		publicacaoID,
	).Scan(&id); erro != nil {
		return erro
	}

	var curtidaExiste bool
	if erro = transacao.QueryRow(
		"SELECT EXISTS (SELECT 1 FROM curtidas_publicacoes WHERE publicacao_id = ? AND usuario_id = ?)",
		publicacaoID,
		usuarioID,
	).Scan(&curtidaExiste); erro != nil {
		return erro
	}

	if curtir && !curtidaExiste {
		if _, erro = transacao.Exec(
			"INSERT INTO curtidas_publicacoes (publicacao_id, usuario_id) VALUES (?, ?)",
			publicacaoID,
			usuarioID,
		); erro != nil {
			return erro
		}
		if _, erro = transacao.Exec(
			"UPDATE publicacoes SET curtidas = curtidas + 1 WHERE id = ?",
			publicacaoID,
		); erro != nil {
			return erro
		}
	} else if !curtir && curtidaExiste {
		if _, erro = transacao.Exec(
			"DELETE FROM curtidas_publicacoes WHERE publicacao_id = ? AND usuario_id = ?",
			publicacaoID,
			usuarioID,
		); erro != nil {
			return erro
		}
		if _, erro = transacao.Exec(
			"UPDATE publicacoes SET curtidas = CASE WHEN curtidas > 0 THEN curtidas - 1 ELSE 0 END WHERE id = ?",
			publicacaoID,
		); erro != nil {
			return erro
		}
	}

	return transacao.Commit()
}

// Pesquisar procura publicacoes por texto (titulo ou conteudo) ou, quando hashtag e verdadeiro,
// pela hashtag exata (#tag nao casa com #tagmaior).
func (repositorio Publicacoes) Pesquisar(viewerID uint64, termo string, hashtag bool, ordem string, limite, deslocamento uint64) ([]modelos.Publicacao, error) {
	var filtro string
	argumentos := []interface{}{viewerID, viewerID}

	if hashtag {
		filtro = "(p.conteudo regexp ? or p.titulo regexp ?)"
		expressao := `(^|[^[:alnum:]_])#` + regexp.QuoteMeta(termo) + `($|[^[:alnum:]_])`
		argumentos = append(argumentos, expressao, expressao)
	} else {
		filtro = "(p.titulo like ? or p.conteudo like ?)"
		padrao := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(termo) + "%"
		argumentos = append(argumentos, padrao, padrao)
	}
	argumentos = append(argumentos, viewerID, viewerID, limite, deslocamento)

	// "populares": curtidas e comentários pesam igual (como no Em alta), sem limite de data
	ordenacao := "p.id desc"
	if ordem == "populares" {
		ordenacao = "(p.curtidas + (select count(*) from comentarios c where c.publicacao_id = p.id)) desc, p.id desc"
	}

	linhas, erro := repositorio.db.Query(`
		select p.*, u.nick, u.suspenso_em is not null, `+colunaFotoVersao+`,
			(select count(*) from comentarios c where c.publicacao_id = p.id),
			`+colunaTemImagem+`,
			EXISTS (
				SELECT 1 FROM curtidas_publicacoes cp
				WHERE cp.publicacao_id = p.id AND cp.usuario_id = ?
			) AS curtido_por_usuario,
			EXISTS (SELECT 1 FROM salvos sv WHERE sv.publicacao_id = p.id AND sv.usuario_id = ?) AS salvo_por_usuario
		from publicacoes p
		join usuarios u on u.id = p.autor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where `+filtro+` and `+postVisivel+`
		order by `+ordenacao+`
		limit ? offset ?`, argumentos...)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	publicacoes := []modelos.Publicacao{}
	for linhas.Next() {
		var publicacao modelos.Publicacao
		var editadaEm sql.NullTime

		if erro = linhas.Scan(
			&publicacao.ID,
			&publicacao.Titulo,
			&publicacao.Conteudo,
			&publicacao.AutorID,
			&publicacao.Curtidas,
			&publicacao.CriadaEm,
			&editadaEm,
			&publicacao.AutorNick,
			&publicacao.AutorSuspenso,
			&publicacao.AutorFotoVersao,
			&publicacao.Comentarios,
			&publicacao.TemImagem,
			&publicacao.CurtidoPorUsuario,
			&publicacao.SalvoPorUsuario,
		); erro != nil {
			return nil, erro
		}

		publicacao.Editado = editadaEm.Valid
		publicacoes = append(publicacoes, publicacao)
	}

	return publicacoes, linhas.Err()
}

// PodeVer informa se o visitante pode ver a publicação (existe e o perfil é público, próprio ou seguido)
func (repositorio Publicacoes) PodeVer(publicacaoID, visitanteID uint64) (bool, error) {
	var visivel bool
	erro := repositorio.db.QueryRow(`
		select 1 from publicacoes p
		inner join usuarios u on u.id = p.autor_id
		where p.id = ? and `+postVisivel, publicacaoID, visitanteID, visitanteID).Scan(&visivel)
	if erro == sql.ErrNoRows {
		return false, nil
	}
	return visivel, erro
}
