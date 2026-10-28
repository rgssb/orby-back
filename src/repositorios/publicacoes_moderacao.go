package repositorios

import "errors"

// ErrContaAtiva impede a remoção administrativa de publicações de contas ativas.
var ErrContaAtiva = errors.New("Administradores não podem excluir publicações de contas ativas")

// DeletarDeContaSuspensa mantém a conta bloqueada durante a verificação e a exclusão.
func (repositorio Publicacoes) DeletarDeContaSuspensa(publicacaoID uint64) error {
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()

	var suspensa bool
	if erro = tx.QueryRow(`select u.suspenso_em is not null
		from publicacoes p join usuarios u on u.id = p.autor_id
		where p.id = ? for update`, publicacaoID).Scan(&suspensa); erro != nil {
		return erro
	}
	if !suspensa {
		return ErrContaAtiva
	}
	linhas, erro := tx.Query("select id from imagens_extras_publicacoes where publicacao_id = ?", publicacaoID)
	if erro != nil {
		return erro
	}
	var extras []uint64
	for linhas.Next() {
		var id uint64
		if erro = linhas.Scan(&id); erro != nil {
			linhas.Close()
			return erro
		}
		extras = append(extras, id)
	}
	erro = linhas.Err()
	linhas.Close()
	if erro != nil {
		return erro
	}
	if _, erro = tx.Exec("delete from publicacoes where id in (select publicacao_id from reposts where original_id = ?)", publicacaoID); erro != nil {
		return erro
	}
	if _, erro = tx.Exec("delete from publicacoes where id = ?", publicacaoID); erro != nil {
		return erro
	}
	if erro = tx.Commit(); erro != nil {
		return erro
	}
	return repositorio.removerArquivosDeImagens(publicacaoID, extras)
}
