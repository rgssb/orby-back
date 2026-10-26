package controllers

import (
	"API/src/autenticacao"
	"API/src/banco"
	"API/src/repositorios"
	"API/src/respostas"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func paraBloqueio(w http.ResponseWriter, r *http.Request) (eu, outro uint64, repositorio *repositorios.Bloqueios, fechar func(), ok bool) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}
	outro, erro = strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	return eu, outro, repositorios.NovoRepositorioDeBloqueios(db), func() { db.Close() }, true
}

// BloquearUsuario bloqueia outra pessoa: nenhum dos dois pode seguir ou conversar com o outro
func BloquearUsuario(w http.ResponseWriter, r *http.Request) {
	eu, outro, repositorio, fechar, ok := paraBloqueio(w, r)
	if !ok {
		return
	}
	defer fechar()

	if eu == outro {
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possível bloquear você mesmo"))
		return
	}
	if erro := repositorio.Bloquear(eu, outro); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// DesbloquearUsuario remove o bloqueio feito pelo usuário autenticado
func DesbloquearUsuario(w http.ResponseWriter, r *http.Request) {
	eu, outro, repositorio, fechar, ok := paraBloqueio(w, r)
	if !ok {
		return
	}
	defer fechar()

	if erro := repositorio.Desbloquear(eu, outro); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// SituacaoDeBloqueio informa se há bloqueio entre o usuário autenticado e outra pessoa
func SituacaoDeBloqueio(w http.ResponseWriter, r *http.Request) {
	eu, outro, repositorio, fechar, ok := paraBloqueio(w, r)
	if !ok {
		return
	}
	defer fechar()

	bloqueei, bloqueado, erro := repositorio.Situacao(eu, outro)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, map[string]bool{"bloqueei": bloqueei, "bloqueado": bloqueado})
}
