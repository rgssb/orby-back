package repositorios

import (
	"API/src/modelos"
	"database/sql"
	"errors"
)

// Salvar guarda uma publicação nos salvos do usuário
func (repositorio Publicacoes) Salvar(usuarioID, publicacaoID uint64) error {
	_, erro := repositorio.db.Exec("insert ignore into salvos (usuario_id, publicacao_id) values (?, ?)", usuarioID, publicacaoID)
	return erro
}

// RemoverSalvo tira uma publicação dos salvos do usuário
func (repositorio Publicacoes) RemoverSalvo(usuarioID, publicacaoID uint64) error {
	_, erro := repositorio.db.Exec("delete from salvos where usuario_id = ? and publicacao_id = ?", usuarioID, publicacaoID)
	return erro
}

// BuscarSalvos traz as publicações salvas pelo usuário, da mais recentemente salva para a mais antiga
func (repositorio Publicacoes) BuscarSalvos(usuarioID, limite, deslocamento uint64) ([]modelos.Publicacao, error) {
	linhas, erro := repositorio.db.Query(`
		select p.*, u.nick, `+colunaFotoVersao+`,
			(select count(*) from comentarios c where c.publicacao_id = p.id),
			`+colunaTemImagem+`,
			EXISTS (SELECT 1 FROM curtidas_publicacoes cp WHERE cp.publicacao_id = p.id AND cp.usuario_id = ?) AS curtido_por_usuario,
			true AS salvo_por_usuario
		from salvos sv
		join publicacoes p on p.id = sv.publicacao_id
		join usuarios u on u.id = p.autor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where sv.usuario_id = ? and `+postVisivel+`
		order by sv.criadoEm desc, p.id desc
		limit ? offset ?`, usuarioID, usuarioID, usuarioID, usuarioID, limite, deslocamento)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	publicacoes := []modelos.Publicacao{}
	for linhas.Next() {
		var publicacao modelos.Publicacao
		var editadaEm sql.NullTime
		if erro = linhas.Scan(
			&publicacao.ID, &publicacao.Titulo, &publicacao.Conteudo, &publicacao.AutorID, &publicacao.Curtidas,
			&publicacao.CriadaEm, &editadaEm, &publicacao.AutorNick, &publicacao.AutorFotoVersao,
			&publicacao.Comentarios, &publicacao.TemImagem, &publicacao.CurtidoPorUsuario, &publicacao.SalvoPorUsuario,
		); erro != nil {
			return nil, erro
		}
		publicacao.Editado = editadaEm.Valid
		publicacoes = append(publicacoes, publicacao)
	}
	return publicacoes, linhas.Err()
}

// Privacidade devolve se o perfil é privado, se o visitante o segue e se já pediu para seguir
func (repositorio usuarios) Privacidade(alvoID, visitanteID uint64) (privado, segue, pedido bool, erro error) {
	erro = repositorio.db.QueryRow(`
		select u.privado,
			exists(select 1 from seguidores where usuario_id = u.id and seguidor_id = ?),
			exists(select 1 from pedidos_seguir where alvo_id = u.id and solicitante_id = ?)
		from usuarios u where u.id = ?`, visitanteID, visitanteID, alvoID,
	).Scan(&privado, &segue, &pedido)
	if errors.Is(erro, sql.ErrNoRows) {
		return false, false, false, nil
	}
	return
}

// DefinirPrivacidade torna o perfil privado ou público; ao abrir o perfil, os pedidos pendentes são aceitos
func (repositorio usuarios) DefinirPrivacidade(usuarioID uint64, privado bool) error {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer transacao.Rollback()

	if _, erro = transacao.Exec("update usuarios set privado = ? where id = ?", privado, usuarioID); erro != nil {
		return erro
	}
	if !privado {
		if _, erro = transacao.Exec(
			"insert ignore into seguidores (usuario_id, seguidor_id) select alvo_id, solicitante_id from pedidos_seguir where alvo_id = ?", usuarioID,
		); erro != nil {
			return erro
		}
		if _, erro = transacao.Exec("update notificacoes set tipo = 'seguidor' where usuario_id = ? and tipo = 'pedido_seguir'", usuarioID); erro != nil {
			return erro
		}
		if _, erro = transacao.Exec("delete from pedidos_seguir where alvo_id = ?", usuarioID); erro != nil {
			return erro
		}
	}
	return transacao.Commit()
}

// PedirParaSeguir registra um pedido de seguimento a um perfil privado
func (repositorio usuarios) PedirParaSeguir(alvoID, solicitanteID uint64) error {
	_, erro := repositorio.db.Exec("insert ignore into pedidos_seguir (solicitante_id, alvo_id) values (?, ?)", solicitanteID, alvoID)
	return erro
}

// CancelarPedido remove o pedido de seguimento (cancelado por quem pediu ou recusado pelo dono do perfil)
func (repositorio usuarios) CancelarPedido(alvoID, solicitanteID uint64) error {
	_, erro := repositorio.db.Exec("delete from pedidos_seguir where alvo_id = ? and solicitante_id = ?", alvoID, solicitanteID)
	return erro
}

// AceitarPedido transforma o pedido em seguimento; devolve false se não havia pedido
func (repositorio usuarios) AceitarPedido(alvoID, solicitanteID uint64) (bool, error) {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return false, erro
	}
	defer transacao.Rollback()

	resultado, erro := transacao.Exec("delete from pedidos_seguir where alvo_id = ? and solicitante_id = ?", alvoID, solicitanteID)
	if erro != nil {
		return false, erro
	}
	if linhas, _ := resultado.RowsAffected(); linhas == 0 {
		return false, nil
	}
	if _, erro = transacao.Exec("insert ignore into seguidores (usuario_id, seguidor_id) values (?, ?)", alvoID, solicitanteID); erro != nil {
		return false, erro
	}
	return true, transacao.Commit()
}

// BuscarPedidos lista quem pediu para seguir o usuário, do mais recente ao mais antigo
func (repositorio usuarios) BuscarPedidos(alvoID uint64) ([]modelos.Usuario, error) {
	linhas, erro := repositorio.db.Query(`
		select u.id, u.nome, u.nick, u.criadoEm, `+colunaFotoVersao+`
		from pedidos_seguir p join usuarios u on u.id = p.solicitante_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where p.alvo_id = ? order by p.criadoEm desc`, alvoID)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	pedidos := []modelos.Usuario{}
	for linhas.Next() {
		var usuario modelos.Usuario
		if erro = linhas.Scan(&usuario.ID, &usuario.Nome, &usuario.Nick, &usuario.CriadoEm, &usuario.FotoVersao); erro != nil {
			return nil, erro
		}
		pedidos = append(pedidos, usuario)
	}
	return pedidos, linhas.Err()
}
