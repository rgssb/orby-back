package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
)

func PreferenciasDePrivacidade(w http.ResponseWriter, r *http.Request) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	if r.Method == http.MethodGet {
		p, erro := repositorios.PreferenciasDePrivacidade(db, eu)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, p)
		return
	}
	var p repositorios.PrivacidadeInteracoes
	if erro = json.NewDecoder(r.Body).Decode(&p); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, repositorios.ErrPreferenciaPrivacidadeInvalida)
		return
	}
	if erro = repositorios.SalvarPrivacidade(db, eu, p); errors.Is(erro, repositorios.ErrPreferenciaPrivacidadeInvalida) {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	} else if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
