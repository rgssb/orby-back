package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"

	"github.com/gorilla/mux"
)

// Denunciar registra a denúncia de um perfil, publicação ou comentário
func Denunciar(w http.ResponseWriter, r *http.Request) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}
	var corpo struct {
		Tipo   string `json:"tipo"`
		AlvoID uint64 `json:"alvoId"`
		Motivo string `json:"motivo"`
	}
	if erro = json.NewDecoder(r.Body).Decode(&corpo); erro != nil || corpo.AlvoID == 0 {
		respostas.Erro(w, http.StatusBadRequest, repositorios.ErrDenunciaInvalida)
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	switch erro = repositorios.NovoRepositorioDeDenuncias(db).Registrar(eu, corpo.Tipo, corpo.AlvoID, corpo.Motivo); {
	case erro == nil:
		respostas.JSON(w, http.StatusNoContent, nil)
	case errors.Is(erro, repositorios.ErrDenunciaInvalida), errors.Is(erro, repositorios.ErrDenunciaPropria):
		respostas.Erro(w, http.StatusBadRequest, erro)
	case errors.Is(erro, repositorios.ErrDenunciaNaoAchada):
		respostas.Erro(w, http.StatusNotFound, erro)
	case errors.Is(erro, repositorios.ErrDenunciasDemais):
		respostas.Erro(w, http.StatusTooManyRequests, erro)
	default:
		respostas.Erro(w, http.StatusInternalServerError, erro)
	}
}

// exigirAdmin devolve o repositório quando quem chama é administrador; caso contrário já responde e devolve nil
func exigirAdmin(w http.ResponseWriter, r *http.Request) (*repositorios.Denuncias, func()) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return nil, nil
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return nil, nil
	}
	repositorio := repositorios.NovoRepositorioDeDenuncias(db)
	admin, erro := repositorio.EhAdmin(eu)
	if erro != nil {
		db.Close()
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return nil, nil
	}
	if !admin {
		db.Close()
		respostas.Erro(w, http.StatusForbidden, repositorios.ErrSemPermissaoDeAdmin)
		return nil, nil
	}
	return repositorio, func() { db.Close() }
}

// ListarPerfisDenunciados mostra à administração os perfis em alerta ou em revisão
func ListarPerfisDenunciados(w http.ResponseWriter, r *http.Request) {
	repositorio, fechar := exigirAdmin(w, r)
	if repositorio == nil {
		return
	}
	defer fechar()

	perfis, erro := repositorio.ListarPerfis()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, perfis)
}

// ResolverDenuncias arquiva as denúncias abertas de um perfil
func ResolverDenuncias(w http.ResponseWriter, r *http.Request) {
	repositorio, fechar := exigirAdmin(w, r)
	if repositorio == nil {
		return
	}
	defer fechar()

	donoID, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return
	}
	if erro = repositorio.Resolver(donoID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
