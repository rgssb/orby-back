package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
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

	donoID, erro := repositorios.NovoRepositorioDeDenuncias(db).RegistrarComDono(eu, corpo.Tipo, corpo.AlvoID, corpo.Motivo)
	switch {
	case erro == nil:
		respostas.JSON(w, http.StatusOK, map[string]uint64{"usuarioId": donoID})
	case errors.Is(erro, repositorios.ErrJaDenunciado):
		respostas.Erro(w, http.StatusConflict, erro)
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

// exigirAdmin devolve a conexão quando quem chama é administrador; caso contrário já responde e devolve nil
func exigirAdmin(w http.ResponseWriter, r *http.Request) *sql.DB {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return nil
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return nil
	}
	admin, erro := repositorios.EhAdmin(db, eu)
	if erro != nil {
		db.Close()
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return nil
	}
	if !admin {
		db.Close()
		respostas.Erro(w, http.StatusForbidden, repositorios.ErrSemPermissaoDeAdmin)
		return nil
	}
	return db
}

// SouAdmin diz ao navegador se mostra as opções de administrador
func SouAdmin(w http.ResponseWriter, r *http.Request) {
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
	admin, erro := repositorios.EhAdmin(db, eu)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, map[string]bool{"admin": admin})
}

// PainelDaAdministracao reúne perfis denunciados, contas suspensas e estatísticas
func PainelDaAdministracao(w http.ResponseWriter, r *http.Request) {
	db := exigirAdmin(w, r)
	if db == nil {
		return
	}

	defer db.Close()

	perfis, erro := repositorios.NovoRepositorioDeDenuncias(db).ListarPerfis()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	administracao := repositorios.NovoRepositorioDeAdministracao(db)
	suspensos, erro := administracao.ListarSuspensos()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	estatisticas, erro := administracao.Estatisticas()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, map[string]any{"denunciados": perfis, "suspensos": suspensos, "estatisticas": estatisticas})
}

// ResolverDenuncias arquiva as denúncias abertas de um perfil
func ResolverDenuncias(w http.ResponseWriter, r *http.Request) {
	db := exigirAdmin(w, r)
	if db == nil {
		return
	}
	defer db.Close()

	donoID, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return
	}
	if erro = repositorios.NovoRepositorioDeDenuncias(db).Resolver(donoID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

func mudarSuspensao(w http.ResponseWriter, r *http.Request, suspender bool) {
	db := exigirAdmin(w, r)
	if db == nil {
		return
	}
	defer db.Close()

	alvoID, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return
	}
	administracao := repositorios.NovoRepositorioDeAdministracao(db)
	if suspender {
		erro = administracao.Suspender(alvoID)
	} else {
		erro = administracao.Reativar(alvoID)
	}
	switch {
	case erro == nil:
		eu, _ := autenticacao.ExtrairUsuarioID(r)
		log.Printf("administração: usuário %d mudou a suspensão do usuário %d (suspender=%v)", eu, alvoID, suspender)
		respostas.JSON(w, http.StatusNoContent, nil)
	case errors.Is(erro, repositorios.ErrUsuarioNaoEncontrado):
		respostas.Erro(w, http.StatusNotFound, erro)
	case errors.Is(erro, repositorios.ErrNaoSuspendeAdmin):
		respostas.Erro(w, http.StatusForbidden, erro)
	default:
		respostas.Erro(w, http.StatusInternalServerError, erro)
	}
}

// SuspenderUsuario barra a conta de usar o Orby
func SuspenderUsuario(w http.ResponseWriter, r *http.Request) { mudarSuspensao(w, r, true) }

// ReativarUsuario devolve o acesso à conta
func ReativarUsuario(w http.ResponseWriter, r *http.Request) { mudarSuspensao(w, r, false) }

// ListarAdministracao fornece listagens paginadas somente a administradores.
func ListarAdministracao(w http.ResponseWriter, r *http.Request) {
	tipo := mux.Vars(r)["tipo"]
	switch tipo {
	case "contas", "novas", "publicacoes", "comentarios", "denunciadas", "suspensas":
	default:
		respostas.Erro(w, http.StatusNotFound, errors.New("Listagem não encontrada"))
		return
	}
	deslocamento, erro := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Deslocamento inválido"))
		return
	}
	db := exigirAdmin(w, r)
	if db == nil {
		return
	}
	defer db.Close()
	itens, erro := repositorios.NovoRepositorioDeAdministracao(db).Listar(tipo, r.URL.Query().Get("q"), deslocamento)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if tipo == "publicacoes" || tipo == "comentarios" {
		adminID, erro := autenticacao.ExtrairUsuarioID(r)
		if erro != nil {
			respostas.Erro(w, http.StatusUnauthorized, erro)
			return
		}
		if erro = repositorios.RegistrarListaModeracao(db, adminID, tipo, itens); erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
	}
	respostas.JSON(w, http.StatusOK, itens)
}
