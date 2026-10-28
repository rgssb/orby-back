package controllers

import (
	"errors"
	"net/http"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"

	"github.com/gorilla/mux"
)

func DetalhesDeDenuncias(w http.ResponseWriter, r *http.Request) {
	db := exigirAdmin(w, r)
	if db == nil {
		return
	}
	defer db.Close()
	alvo, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil || alvo == 0 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return
	}
	deslocamento, erro := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Deslocamento inválido"))
		return
	}
	repo := repositorios.NovoRepositorioDeDenuncias(db)
	detalhes, erro := repo.Detalhes(alvo, deslocamento)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	acoes, erro := repo.AcoesRecentes(alvo)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, map[string]interface{}{"denuncias": detalhes, "acoes": acoes})
}
