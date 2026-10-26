package repositorios

import (
	"database/sql"
	"errors"
)

// ErroTokenInvalido indica que o link de redefinicao de senha nao existe ou expirou
var ErroTokenInvalido = errors.New("O link de recuperação é inválido ou expirou")

// BuscarParaRecuperacao traz o ID e o nome do usuario dono do e-mail (ID zero quando nao existe)
func (repositorio usuarios) BuscarParaRecuperacao(email string) (uint64, string, error) {
	var (
		id   uint64
		nome string
	)

	erro := repositorio.db.QueryRow("select id, nome from usuarios where email = ?", email).Scan(&id, &nome)
	if errors.Is(erro, sql.ErrNoRows) {
		return 0, "", nil
	}

	return id, nome, erro
}

// CriarRedefinicaoDeSenha guarda o hash do token de recuperacao, valido por 30 minutos,
// e descarta os tokens anteriores do usuario. Retorna false quando ja existe um token
// criado ha menos de um minuto (evita disparo repetido de e-mails).
func (repositorio usuarios) CriarRedefinicaoDeSenha(usuarioID uint64, tokenHash string) (bool, error) {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return false, erro
	}
	defer transacao.Rollback()

	var recentes int
	if erro = transacao.QueryRow(
		"select count(*) from redefinicoes_senha where usuario_id = ? and criadoEm > date_sub(now(), interval 1 minute)",
		usuarioID,
	).Scan(&recentes); erro != nil {
		return false, erro
	}

	if recentes > 0 {
		return false, nil
	}

	if _, erro = transacao.Exec("delete from redefinicoes_senha where usuario_id = ?", usuarioID); erro != nil {
		return false, erro
	}

	if _, erro = transacao.Exec(
		"insert into redefinicoes_senha (usuario_id, token_hash, expiraEm) values (?, ?, date_add(now(), interval 30 minute))",
		usuarioID, tokenHash,
	); erro != nil {
		return false, erro
	}

	return true, transacao.Commit()
}

// RedefinirSenhaComToken troca a senha do dono do token e invalida todos os seus tokens
func (repositorio usuarios) RedefinirSenhaComToken(tokenHash, senhaComHash string) error {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer transacao.Rollback()

	var usuarioID uint64
	erro = transacao.QueryRow(
		"select usuario_id from redefinicoes_senha where token_hash = ? and expiraEm > now() for update",
		tokenHash,
	).Scan(&usuarioID)
	if errors.Is(erro, sql.ErrNoRows) {
		return ErroTokenInvalido
	}
	if erro != nil {
		return erro
	}

	if _, erro = transacao.Exec("update usuarios set senha = ? where id = ?", senhaComHash, usuarioID); erro != nil {
		return erro
	}

	if _, erro = transacao.Exec("delete from redefinicoes_senha where usuario_id = ?", usuarioID); erro != nil {
		return erro
	}

	return transacao.Commit()
}
