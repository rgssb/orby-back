package controllers

import (
	"database/sql"
	"errors"
	"net/http"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
)

// verificarLeituraPublicacao permite uma leitura administrativa auditada sem liberar interações.
func verificarLeituraPublicacao(w http.ResponseWriter, db *sql.DB, publicacaoID, usuarioID uint64, tipo string) (bool, bool) {
	pode, erro := repositorios.NovoRepositorioDePublicacoes(db).PodeVer(publicacaoID, usuarioID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return false, false
	}
	if pode {
		return true, false
	}
	pode, erro = repositorios.AutorizarLeituraModeracao(db, usuarioID, publicacaoID, tipo)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return false, false
	}
	if !pode {
		respostas.Erro(w, http.StatusNotFound, errors.New("Publicação não encontrada"))
	}
	return pode, pode
}
