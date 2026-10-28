package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
)

// RegistrarAcessoModeracao registra o acesso sem guardar o conteúdo visualizado.
func RegistrarAcessoModeracao(db *sql.DB, adminID, alvoID, publicacaoID uint64, tipo string) error {
	var publicacao interface{}
	if publicacaoID != 0 {
		publicacao = publicacaoID
	}
	_, erro := db.Exec("insert into acessos_moderacao (admin_id, alvo_id, publicacao_id, tipo) values (?, ?, ?, ?)", adminID, alvoID, publicacao, tipo)
	return erro
}

// AutorizarLeituraModeracao concede acesso somente a administradores e registra a leitura.
func AutorizarLeituraModeracao(db *sql.DB, adminID, publicacaoID uint64, tipo string) (bool, error) {
	var alvoID uint64
	erro := db.QueryRow(`
		select p.autor_id from publicacoes p
		join usuarios u on u.id = p.autor_id
		join usuarios admin on admin.id = ? and admin.admin = 1
		where p.id = ? and u.privado = 1`, adminID, publicacaoID).Scan(&alvoID)
	if errors.Is(erro, sql.ErrNoRows) {
		return false, nil
	}
	if erro != nil {
		return false, erro
	}
	if erro = RegistrarAcessoModeracao(db, adminID, alvoID, publicacaoID, tipo); erro != nil {
		return false, erro
	}
	return true, nil
}

// RegistrarListaModeracao registra os conteúdos privados incluídos numa listagem administrativa.
func RegistrarListaModeracao(db *sql.DB, adminID uint64, tipo string, itens []modelos.ItemAdministrativo) error {
	tx, erro := db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	vistos := make(map[[2]uint64]bool)
	for _, item := range itens {
		chave := [2]uint64{item.AlvoPrivadoID, item.ID}
		if item.AlvoPrivadoID == 0 || vistos[chave] {
			continue
		}
		vistos[chave] = true
		if _, erro = tx.Exec("insert into acessos_moderacao (admin_id, alvo_id, publicacao_id, tipo) values (?, ?, ?, ?)",
			adminID, item.AlvoPrivadoID, item.ID, "lista_"+tipo); erro != nil {
			return erro
		}
	}
	return tx.Commit()
}
