package repositorios

import (
	"strings"

	"orby/api/src/modelos"
)

// PodeRepostar informa se a publicação existe e pode ser repostada: contas privadas só repostam o próprio conteúdo
func (repositorio Publicacoes) PodeRepostar(publicacaoID, usuarioID uint64) (bool, error) {
	var total int
	erro := repositorio.db.QueryRow(`
		select count(*) from publicacoes p inner join usuarios u on u.id = p.autor_id
		where p.id = ? and (u.privado = 0 or u.id = ?)`, publicacaoID, usuarioID).Scan(&total)
	return total > 0, erro
}

// OriginalDoRepost devolve a publicação original quando o ID é de um repost (reposts de reposts apontam para a origem)
func (repositorio Publicacoes) OriginalDoRepost(publicacaoID uint64) (uint64, error) {
	var original uint64
	erro := repositorio.db.QueryRow("select coalesce((select original_id from reposts where publicacao_id = ?), ?)", publicacaoID, publicacaoID).Scan(&original)
	return original, erro
}

// RepostSimples devolve o ID do repost sem comentário do usuário para o original (0 se não houver)
func (repositorio Publicacoes) RepostSimples(usuarioID, originalID uint64) (uint64, error) {
	var id uint64
	erro := repositorio.db.QueryRow(`
		select coalesce(max(p.id), 0) from publicacoes p inner join reposts r on r.publicacao_id = p.id
		where p.autor_id = ? and r.original_id = ? and p.conteudo = ''`, usuarioID, originalID).Scan(&id)
	return id, erro
}

// Repostar cria uma publicação que aponta para a original; o comentário é opcional
func (repositorio Publicacoes) Repostar(usuarioID, originalID uint64, comentario string) (uint64, error) {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return 0, erro
	}
	defer transacao.Rollback()

	resultado, erro := transacao.Exec("insert into publicacoes (titulo, conteudo, autor_id) values ('', ?, ?)", comentario, usuarioID)
	if erro != nil {
		return 0, erro
	}
	id, erro := resultado.LastInsertId()
	if erro != nil {
		return 0, erro
	}
	if _, erro = transacao.Exec("insert into reposts (publicacao_id, original_id) values (?, ?)", id, originalID); erro != nil {
		return 0, erro
	}
	return uint64(id), transacao.Commit()
}

// AnexarRepostagens preenche, em cada publicação, os dados do original (quando é repost), o total de reposts e se o usuário já repostou
func (repositorio Publicacoes) AnexarRepostagens(publicacoes []modelos.Publicacao, visitanteID uint64) error {
	if len(publicacoes) == 0 {
		return nil
	}
	ids := make([]interface{}, len(publicacoes))
	posicoes := make(map[uint64]int, len(publicacoes))
	for i, p := range publicacoes {
		ids[i] = p.ID
		posicoes[p.ID] = i
	}
	marcadores := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")

	linhas, erro := repositorio.db.Query(`
		select r.original_id, count(*), coalesce(max(p.autor_id = ? and p.conteudo = ''), 0)
		from reposts r inner join publicacoes p on p.id = r.publicacao_id
		where r.original_id in (`+marcadores+`) group by r.original_id`, append([]interface{}{visitanteID}, ids...)...)
	if erro != nil {
		return erro
	}
	for linhas.Next() {
		var id, total uint64
		var meu bool
		if erro = linhas.Scan(&id, &total, &meu); erro != nil {
			linhas.Close()
			return erro
		}
		publicacoes[posicoes[id]].Reposts = total
		publicacoes[posicoes[id]].RepostadoPorUsuario = meu
	}
	linhas.Close()

	linhas, erro = repositorio.db.Query(`
		select r.publicacao_id, o.id, o.autor_id, u.nick, `+colunaFotoVersao+`, o.titulo, o.conteudo, o.criadoEm,
			exists(select 1 from imagens_publicacoes ip where ip.publicacao_id = o.id),
			`+postVisivel+`
		from reposts r
		inner join publicacoes o on o.id = r.original_id
		inner join usuarios u on u.id = o.autor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where r.publicacao_id in (`+marcadores+`)`, append([]interface{}{visitanteID, visitanteID}, ids...)...)
	if erro != nil {
		return erro
	}
	defer linhas.Close()
	for linhas.Next() {
		var repostID uint64
		var original modelos.PublicacaoOriginal
		var visivel bool
		if erro = linhas.Scan(&repostID, &original.ID, &original.AutorID, &original.AutorNick, &original.AutorFotoVersao,
			&original.Titulo, &original.Conteudo, &original.CriadaEm, &original.TemImagem, &visivel); erro != nil {
			return erro
		}
		if !visivel {
			original = modelos.PublicacaoOriginal{ID: original.ID, Indisponivel: true}
		}
		publicacoes[posicoes[repostID]].RepostDe = &original
	}
	return linhas.Err()
}
