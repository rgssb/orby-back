package controllers

import (
	"database/sql"
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

const limiteDeNotificacoes = 30

// BuscarNotificacoes lista as notificacoes mais recentes do usuario autenticado
func BuscarNotificacoes(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
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

	limite, _ := strconv.ParseUint(r.URL.Query().Get("limite"), 10, 64)
	if limite == 0 {
		limite = limiteDeNotificacoes
	}
	if limite > 100 {
		limite = 100
	}
	deslocamento, _ := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)

	notificacoes, erro := repositorios.NovoRepositorioDeNotificacoes(db).Listar(usuarioID, limite, deslocamento)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusOK, notificacoes)
}

// ContarNotificacoesNaoLidas devolve o total de notificacoes ainda nao vistas
func ContarNotificacoesNaoLidas(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
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

	total, erro := repositorios.NovoRepositorioDeNotificacoes(db).ContarNaoLidas(usuarioID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusOK, map[string]uint64{"total": total})
}

// MarcarNotificacoesComoLidas marca todas as notificacoes do usuario como lidas
func MarcarNotificacoesComoLidas(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
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

	if erro = repositorios.NovoRepositorioDeNotificacoes(db).MarcarTodasComoLidas(usuarioID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// registrarNotificacao executa a gravacao de uma notificacao sem interromper a acao principal:
// uma falha ao avisar nao deve impedir a curtida ou o comentario.
func registrarNotificacao(db *sql.DB, gravar func(*repositorios.Notificacoes) error) {
	if erro := gravar(repositorios.NovoRepositorioDeNotificacoes(db)); erro != nil {
		log.Printf("notificacao: %v", erro)
	}
}

// ExcluirNotificacao remove uma notificacao do usuario autenticado
func ExcluirNotificacao(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	notificacaoID, erro := strconv.ParseUint(mux.Vars(r)["notificacaoId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	existia, erro := repositorios.NovoRepositorioDeNotificacoes(db).Excluir(notificacaoID, usuarioID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if !existia {
		respostas.Erro(w, http.StatusNotFound, errors.New("Notificação não encontrada"))
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}
