package repositorios

import (
	"database/sql"
	"orby/api/src/modelos"
	"regexp"
)

var expressaoDeMencao = regexp.MustCompile(`(?:^|[^\w@])@([A-Za-z0-9_]{1,50})`)

// Notificacoes representa um repositorio de notificacoes
type Notificacoes struct {
	db *sql.DB
}

// NovoRepositorioDeNotificacoes cria um repositorio de notificacoes
func NovoRepositorioDeNotificacoes(db *sql.DB) *Notificacoes {
	return &Notificacoes{db: db}
}

// NotificarCurtidaDePublicacao avisa o autor da publicacao (nao avisa a si mesmo e nao repete)
func (repositorio Notificacoes) NotificarCurtidaDePublicacao(publicacaoID, atorID uint64) error {
	_, erro := repositorio.db.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo, publicacao_id)
		select p.autor_id, ?, ?, p.id from publicacoes p
		where p.id = ? and p.autor_id <> ?
		and not exists (
			select 1 from notificacoes n
			where n.ator_id = ? and n.tipo = ? and n.publicacao_id = p.id
		)`,
		atorID, modelos.NotificacaoCurtidaPublicacao, publicacaoID, atorID,
		atorID, modelos.NotificacaoCurtidaPublicacao,
	)
	return erro
}

// NotificarComentario avisa o autor da publicacao sobre um novo comentario
func (repositorio Notificacoes) NotificarComentario(publicacaoID, comentarioID, atorID uint64) error {
	_, erro := repositorio.db.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo, publicacao_id, comentario_id)
		select p.autor_id, ?, ?, p.id, ? from publicacoes p
		where p.id = ? and p.autor_id <> ?`,
		atorID, modelos.NotificacaoComentario, comentarioID, publicacaoID, atorID,
	)
	return erro
}

// NotificarResposta avisa o autor do comentario respondido. O dono da publicacao nao recebe este aviso
// porque ja e avisado do comentario; o aviso aponta para o comentario principal da conversa (raizID).
func (repositorio Notificacoes) NotificarResposta(paiID, raizID, atorID uint64) error {
	_, erro := repositorio.db.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo, publicacao_id, comentario_id)
		select c.autor_id, ?, ?, c.publicacao_id, ? from comentarios c
		inner join publicacoes p on p.id = c.publicacao_id
		where c.id = ? and c.autor_id <> ? and c.autor_id <> p.autor_id`,
		atorID, modelos.NotificacaoResposta, raizID, paiID, atorID,
	)
	return erro
}

// NotificarCurtidaDeComentario avisa o autor do comentario (nao avisa a si mesmo e nao repete)
func (repositorio Notificacoes) NotificarCurtidaDeComentario(comentarioID, atorID uint64) error {
	_, erro := repositorio.db.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo, publicacao_id, comentario_id)
		select c.autor_id, ?, ?, c.publicacao_id, c.id from comentarios c
		where c.id = ? and c.autor_id <> ?
		and not exists (
			select 1 from notificacoes n
			where n.ator_id = ? and n.tipo = ? and n.comentario_id = c.id
		)`,
		atorID, modelos.NotificacaoCurtidaComentario, comentarioID, atorID,
		atorID, modelos.NotificacaoCurtidaComentario,
	)
	return erro
}

// NotificarSeguidor avisa o usuario seguido (nao repete enquanto o aviso anterior existir)
func (repositorio Notificacoes) NotificarSeguidor(usuarioID, seguidorID uint64) error {
	_, erro := repositorio.db.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo)
		select ?, ?, ? from dual
		where ? <> ? and not exists (
			select 1 from notificacoes n
			where n.usuario_id = ? and n.ator_id = ? and n.tipo = ?
		)`,
		usuarioID, seguidorID, modelos.NotificacaoSeguidor,
		usuarioID, seguidorID,
		usuarioID, seguidorID, modelos.NotificacaoSeguidor,
	)
	return erro
}

// NotificarPedidoParaSeguir avisa o dono de um perfil privado sobre um novo pedido
func (repositorio Notificacoes) NotificarPedidoParaSeguir(usuarioID, solicitanteID uint64) error {
	_, erro := repositorio.db.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo)
		select ?, ?, ? from dual
		where ? <> ? and not exists (
			select 1 from notificacoes n
			where n.usuario_id = ? and n.ator_id = ? and n.tipo = ?
		)`,
		usuarioID, solicitanteID, modelos.NotificacaoPedidoSeguir,
		usuarioID, solicitanteID,
		usuarioID, solicitanteID, modelos.NotificacaoPedidoSeguir,
	)
	return erro
}

// RemoverPedidoParaSeguir apaga o aviso do pedido (aceito, recusado ou cancelado)
func (repositorio Notificacoes) RemoverPedidoParaSeguir(usuarioID, solicitanteID uint64) error {
	_, erro := repositorio.db.Exec(
		"delete from notificacoes where usuario_id = ? and ator_id = ? and tipo = ?",
		usuarioID, solicitanteID, modelos.NotificacaoPedidoSeguir,
	)
	return erro
}

// RemoverSeguidor apaga o aviso quando o usuario deixa de seguir
func (repositorio Notificacoes) RemoverSeguidor(usuarioID, seguidorID uint64) error {
	_, erro := repositorio.db.Exec(
		"delete from notificacoes where usuario_id = ? and ator_id = ? and tipo = ?",
		usuarioID, seguidorID, modelos.NotificacaoSeguidor,
	)
	return erro
}

// RemoverCurtidaDePublicacao apaga o aviso quando a curtida e desfeita
func (repositorio Notificacoes) RemoverCurtidaDePublicacao(publicacaoID, atorID uint64) error {
	_, erro := repositorio.db.Exec(
		"delete from notificacoes where ator_id = ? and tipo = ? and publicacao_id = ?",
		atorID, modelos.NotificacaoCurtidaPublicacao, publicacaoID,
	)
	return erro
}

// RemoverCurtidaDeComentario apaga o aviso quando a curtida e desfeita
func (repositorio Notificacoes) RemoverCurtidaDeComentario(comentarioID, atorID uint64) error {
	_, erro := repositorio.db.Exec(
		"delete from notificacoes where ator_id = ? and tipo = ? and comentario_id = ?",
		atorID, modelos.NotificacaoCurtidaComentario, comentarioID,
	)
	return erro
}

// Listar traz as notificacoes mais recentes do usuario (com deslocamento para paginar)
func (repositorio Notificacoes) Listar(usuarioID, limite, deslocamento uint64) ([]modelos.Notificacao, error) {
	linhas, erro := repositorio.db.Query(`
		select n.id, n.tipo, n.ator_id, u.nick, `+colunaFotoVersao+`, coalesce(n.publicacao_id, 0), coalesce(p.titulo, ''),
			coalesce(n.comentario_id, 0), n.lida, n.criadoEm
		from notificacoes n
		inner join usuarios u on u.id = n.ator_id
		left join publicacoes p on p.id = n.publicacao_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where n.usuario_id = ?
		order by n.id desc
		limit ? offset ?`, usuarioID, limite, deslocamento,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	notificacoes := []modelos.Notificacao{}
	for linhas.Next() {
		var n modelos.Notificacao
		if erro = linhas.Scan(
			&n.ID, &n.Tipo, &n.AtorID, &n.AtorNick, &n.AtorFotoVersao,
			&n.PublicacaoID, &n.PublicacaoTitulo, &n.ComentarioID, &n.Lida, &n.CriadaEm,
		); erro != nil {
			return nil, erro
		}
		notificacoes = append(notificacoes, n)
	}

	return notificacoes, linhas.Err()
}

// ContarNaoLidas devolve quantas notificacoes o usuario ainda nao viu
func (repositorio Notificacoes) ContarNaoLidas(usuarioID uint64) (uint64, error) {
	var total uint64
	erro := repositorio.db.QueryRow(
		"select count(*) from notificacoes where usuario_id = ? and lida = false", usuarioID,
	).Scan(&total)
	return total, erro
}

// MarcarTodasComoLidas marca como lidas todas as notificacoes do usuario
func (repositorio Notificacoes) MarcarTodasComoLidas(usuarioID uint64) error {
	_, erro := repositorio.db.Exec("update notificacoes set lida = true where usuario_id = ? and lida = false", usuarioID)
	return erro
}

// NotificarMencoes avisa cada usuario citado com @nick (nao avisa o autor nem o dono da publicacao,
// que ja recebe o aviso de comentario). comentarioID zero indica mencao no texto da publicacao.
// Espacos do nick sao ignorados na comparacao: @usuario2 encontra "usuario 2".
func (repositorio Notificacoes) NotificarMencoes(texto string, publicacaoID, comentarioID, atorID uint64) error {
	var comentario interface{}
	filtroDoDono := ""
	if comentarioID != 0 {
		comentario = comentarioID
		filtroDoDono = " and u.id <> (select autor_id from publicacoes where id = ?)"
	}

	consulta := `
		insert into notificacoes (usuario_id, ator_id, tipo, publicacao_id, comentario_id)
		select u.id, ?, ?, ?, ? from usuarios u
		where replace(u.nick, ' ', '') = ? and u.id <> ?` + filtroDoDono + `
		order by u.nick = ? desc limit 1`

	vistos := map[string]bool{}
	for _, achado := range expressaoDeMencao.FindAllStringSubmatch(texto, -1) {
		nick := achado[1]
		if vistos[nick] {
			continue
		}
		vistos[nick] = true

		argumentos := []interface{}{atorID, modelos.NotificacaoMencao, publicacaoID, comentario, nick, atorID}
		if comentarioID != 0 {
			argumentos = append(argumentos, publicacaoID)
		}
		argumentos = append(argumentos, nick)
		if _, erro := repositorio.db.Exec(consulta, argumentos...); erro != nil {
			return erro
		}
	}

	return nil
}

// Excluir apaga uma notificacao; so o dono dela pode fazer isso. Devolve false quando nao existe.
func (repositorio Notificacoes) Excluir(notificacaoID, usuarioID uint64) (bool, error) {
	resultado, erro := repositorio.db.Exec("delete from notificacoes where id = ? and usuario_id = ?", notificacaoID, usuarioID)
	if erro != nil {
		return false, erro
	}

	linhas, erro := resultado.RowsAffected()
	return linhas > 0, erro
}
