package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
)

func PrivacidadeCurtidas(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	if r.Method == http.MethodPut {
		var corpo struct {
			Publica *bool `json:"publica"`
		}
		if erro = json.NewDecoder(r.Body).Decode(&corpo); erro != nil || corpo.Publica == nil {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Preferência inválida"))
			return
		}
		if erro = repositorios.SalvarPrivacidadeCurtidas(db, eu, *corpo.Publica); erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		respostas.JSON(w, http.StatusNoContent, nil)
		return
	}
	publica, erro := repositorios.PrivacidadeCurtidas(db, eu)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, struct {
		Publica bool `json:"publica"`
	}{publica})
}

func CurtidasDoPerfil(w http.ResponseWriter, r *http.Request) {
	eu, alvo, ok := parametroAutenticado(w, r, "usuarioId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	resultado, erro := repositorios.NovoRepositorioDePublicacoes(db).CurtidasDoPerfil(alvo, eu, r.URL.Query().Get("lista") == "1")
	if erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Usuário não encontrado"))
		} else {
			respostas.Erro(w, http.StatusInternalServerError, erro)
		}
		return
	}
	respostas.JSON(w, http.StatusOK, resultado)
}
