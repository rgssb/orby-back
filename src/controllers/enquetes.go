package controllers

import (
	"errors"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"

	"github.com/gorilla/mux"
)

func VotarEnquete(w http.ResponseWriter, r *http.Request) {
	eu, publicacaoID, ok := parametroAutenticado(w, r, "publicacaoId")
	if !ok {
		return
	}
	opcaoID, erro := strconv.ParseUint(mux.Vars(r)["opcaoId"], 10, 64)
	if erro != nil || opcaoID == 0 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Opção inválida"))
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	if !verificarAcessoPublicacao(w, db, publicacaoID, eu) {
		return
	}
	erro = repositorios.NovoRepositorioDePublicacoes(db).VotarEnquete(eu, publicacaoID, opcaoID)
	if erro != nil {
		if errors.Is(erro, repositorios.ErrVotoExistente) || errors.Is(erro, repositorios.ErrEnqueteIndisponivel) {
			respostas.Erro(w, http.StatusConflict, erro)
		} else {
			respostas.Erro(w, http.StatusInternalServerError, erro)
		}
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
