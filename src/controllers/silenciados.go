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

func mudarSilencio(w http.ResponseWriter, r *http.Request, silenciar bool) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}
	outro, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return
	}
	if silenciar && eu == outro {
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possível silenciar você mesmo"))
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDeSilenciados(db)
	if silenciar {
		var existe bool
		if erro = db.QueryRow("select exists(select 1 from usuarios where id = ?)", outro).Scan(&existe); erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		if !existe {
			respostas.Erro(w, http.StatusNotFound, errors.New("Usuário não encontrado"))
			return
		}
		erro = repositorio.Silenciar(eu, outro)
	} else {
		erro = repositorio.DeixarDeSilenciar(eu, outro)
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// SilenciarUsuario esconde as publicações de outra pessoa dos feeds do usuário autenticado
func SilenciarUsuario(w http.ResponseWriter, r *http.Request) {
	mudarSilencio(w, r, true)
}

// DeixarDeSilenciarUsuario volta a mostrar as publicações de outra pessoa nos feeds
func DeixarDeSilenciarUsuario(w http.ResponseWriter, r *http.Request) {
	mudarSilencio(w, r, false)
}
