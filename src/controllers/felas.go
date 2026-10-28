package controllers

import (
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"

	"github.com/gorilla/mux"
)

func GerenciarFelas(w http.ResponseWriter, r *http.Request) {
	dono, erro := autenticacao.ExtrairUsuarioID(r)
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
	repo := repositorios.NovoRepositorioDeFelas(db)
	if r.URL.Path == "/felas" {
		lista, erro := repo.Listar(dono)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, lista)
		return
	}
	membro, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil || membro == 0 {
		respostas.Erro(w, http.StatusBadRequest, repositorios.ErrFelaInvalido)
		return
	}
	switch r.Method {
	case http.MethodGet:
		existe, podeAdicionar, erro := repo.Estado(dono, membro)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, map[string]bool{"fela": existe, "podeAdicionar": podeAdicionar})
		return
	case http.MethodPut:
		erro = repo.Adicionar(dono, membro)
	case http.MethodDelete:
		erro = repo.Remover(dono, membro)
	}
	if errors.Is(erro, repositorios.ErrFelaInvalido) || errors.Is(erro, repositorios.ErrFelasSemSeguimentoMutuo) {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
