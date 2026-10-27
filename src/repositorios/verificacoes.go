package repositorios

import (
	"database/sql"
	"errors"
)

// ErroLinkDeConfirmacaoInvalido indica que o link de confirmação de e-mail não existe ou expirou
var ErroLinkDeConfirmacaoInvalido = errors.New("O link de confirmação é inválido ou expirou")

// BuscarParaConfirmacao traz o ID, o nome e se o e-mail já foi confirmado (ID zero quando não existe)
func (repositorio usuarios) BuscarParaConfirmacao(email string) (id uint64, nome string, confirmado bool, erro error) {
	erro = repositorio.db.QueryRow(
		"select id, nome, email_verificado_em is not null from usuarios where email = ?", email,
	).Scan(&id, &nome, &confirmado)
	if errors.Is(erro, sql.ErrNoRows) {
		return 0, "", false, nil
	}
	return
}

// EmailConfirmado diz se o usuário já confirmou o e-mail
func (repositorio usuarios) EmailConfirmado(usuarioID uint64) (confirmado bool, erro error) {
	erro = repositorio.db.QueryRow("select email_verificado_em is not null from usuarios where id = ?", usuarioID).Scan(&confirmado)
	return
}

// CriarConfirmacaoDeEmail guarda o hash do token, válido por 24 horas, e descarta os anteriores.
// Retorna false quando já existe um token criado há menos de um minuto (evita disparo repetido de e-mails).
func (repositorio usuarios) CriarConfirmacaoDeEmail(usuarioID uint64, tokenHash string) (bool, error) {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return false, erro
	}
	defer transacao.Rollback()

	var recentes int
	if erro = transacao.QueryRow(
		"select count(*) from verificacoes_email where usuario_id = ? and criadoEm > date_sub(now(), interval 1 minute)",
		usuarioID,
	).Scan(&recentes); erro != nil {
		return false, erro
	}
	if recentes > 0 {
		return false, nil
	}

	if _, erro = transacao.Exec("delete from verificacoes_email where usuario_id = ?", usuarioID); erro != nil {
		return false, erro
	}
	if _, erro = transacao.Exec(
		"insert into verificacoes_email (usuario_id, token_hash, expiraEm) values (?, ?, date_add(now(), interval 24 hour))",
		usuarioID, tokenHash,
	); erro != nil {
		return false, erro
	}
	return true, transacao.Commit()
}

// ConfirmarEmailComToken marca o e-mail do dono do token como confirmado e descarta seus tokens
func (repositorio usuarios) ConfirmarEmailComToken(tokenHash string) error {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer transacao.Rollback()

	var usuarioID uint64
	erro = transacao.QueryRow(
		"select usuario_id from verificacoes_email where token_hash = ? and expiraEm > now() for update",
		tokenHash,
	).Scan(&usuarioID)
	if errors.Is(erro, sql.ErrNoRows) {
		return ErroLinkDeConfirmacaoInvalido
	}
	if erro != nil {
		return erro
	}

	if _, erro = transacao.Exec("update usuarios set email_verificado_em = coalesce(email_verificado_em, now()) where id = ?", usuarioID); erro != nil {
		return erro
	}
	if _, erro = transacao.Exec("delete from verificacoes_email where usuario_id = ?", usuarioID); erro != nil {
		return erro
	}
	return transacao.Commit()
}
