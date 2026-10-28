package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
)

// ConsultarDenuncias verifica apenas as denúncias do usuário autenticado.
func ConsultarDenuncias(w http.ResponseWriter, r *http.Request) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}
	var corpo struct {
		Alvos []modelos.AlvoDeDenuncia `json:"alvos"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if erro = json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, repositorios.ErrDenunciaInvalida)
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	alvos, erro := repositorios.NovoRepositorioDeDenuncias(db).Consultar(eu, corpo.Alvos)
	if errors.Is(erro, repositorios.ErrDenunciaInvalida) {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, alvos)
}
