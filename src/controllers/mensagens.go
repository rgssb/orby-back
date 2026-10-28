package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"orby/api/src/seguranca"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// No máximo 30 mensagens por minuto por usuário, contra spam
var limitadorDeMensagens = seguranca.NovoLimitador(30, time.Minute)

func parametroDeMensagens(w http.ResponseWriter, r *http.Request, nome string) (eu, id uint64, ok bool) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return 0, 0, false
	}
	if nome == "" {
		return eu, 0, true
	}
	id, erro = strconv.ParseUint(mux.Vars(r)[nome], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return 0, 0, false
	}
	return eu, id, true
}

func erroDeMensagem(w http.ResponseWriter, erro error) {
	switch {
	case errors.Is(erro, repositorios.ErrUsuarioInexistente), errors.Is(erro, repositorios.ErrMensagemInexistente):
		respostas.Erro(w, http.StatusNotFound, erro)
	case errors.Is(erro, repositorios.ErrMensagemParaSiMesmo), errors.Is(erro, repositorios.ErrRespostaInvalida):
		respostas.Erro(w, http.StatusBadRequest, erro)
	case errors.Is(erro, repositorios.ErrAguardandoAceite), errors.Is(erro, repositorios.ErrBloqueado), errors.Is(erro, repositorios.ErrPedidoEncerrado), errors.Is(erro, repositorios.ErrPrivacidadeInteracao):
		respostas.Erro(w, http.StatusForbidden, erro)
	default:
		respostas.Erro(w, http.StatusInternalServerError, erro)
	}
}

// ListarConversas devolve as conversas e pedidos de conversa do usuário autenticado
func ListarConversas(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroDeMensagens(w, r, "")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	conversas, erro := repositorios.NovoRepositorioDeMensagens(db).Listar(eu)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, conversas)
}

// ContarMensagensNaoLidas devolve o total de mensagens e pedidos novos
func ContarMensagensNaoLidas(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroDeMensagens(w, r, "")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	mensagens, solicitacoes, erro := repositorios.NovoRepositorioDeMensagens(db).ContarNaoLidas(eu)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	grupos, erro := repositorios.NovoRepositorioDeGrupos(db).ContarNaoLidas(eu)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, map[string]uint64{
		"mensagens": mensagens, "solicitacoes": solicitacoes, "grupos": grupos, "total": mensagens + solicitacoes + grupos,
	})
}

// BuscarMensagens abre a conversa com uma pessoa (use ?apos=ID para trazer só as novas)
func BuscarMensagens(w http.ResponseWriter, r *http.Request) {
	eu, outro, ok := parametroDeMensagens(w, r, "usuarioId")
	if !ok {
		return
	}
	apos, _ := strconv.ParseUint(r.URL.Query().Get("apos"), 10, 64)
	antes, _ := strconv.ParseUint(r.URL.Query().Get("antes"), 10, 64)

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	dialogo, erro := repositorios.NovoRepositorioDeMensagens(db).Abrir(eu, outro, apos, antes)
	if erro != nil {
		erroDeMensagem(w, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, dialogo)
}

// EnviarMensagem envia uma mensagem direta; sem seguimento mútuo ela chega como pedido de conversa
func EnviarMensagem(w http.ResponseWriter, r *http.Request) {
	eu, outro, ok := parametroDeMensagens(w, r, "usuarioId")
	if !ok {
		return
	}

	var corpo struct {
		Texto     string `json:"texto"`
		RespondeA uint64 `json:"respondeA"`
		ComImagem bool   `json:"comImagem"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}
	var texto string
	var erro error
	if corpo.ComImagem {
		texto, erro = modelos.PrepararTextoOpcional(corpo.Texto)
	} else {
		texto, erro = modelos.PrepararTexto(corpo.Texto)
	}
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	chave := strconv.FormatUint(eu, 10)
	if limitadorDeMensagens.Bloqueado(chave) {
		respostas.Erro(w, http.StatusTooManyRequests, errors.New("Você está enviando mensagens rápido demais. Aguarde um instante"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	mensagem, erro := repositorios.NovoRepositorioDeMensagens(db).Enviar(eu, outro, strings.TrimSpace(texto), corpo.RespondeA, corpo.ComImagem)
	if erro != nil {
		erroDeMensagem(w, erro)
		return
	}
	limitadorDeMensagens.Registrar(chave)
	respostas.JSON(w, http.StatusCreated, mensagem)
}

// AceitarConversa aceita um pedido de conversa recebido
func AceitarConversa(w http.ResponseWriter, r *http.Request) {
	eu, outro, ok := parametroDeMensagens(w, r, "usuarioId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	aceita, erro := repositorios.NovoRepositorioDeMensagens(db).Aceitar(eu, outro)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if !aceita {
		respostas.Erro(w, http.StatusNotFound, errors.New("Não há pedido de conversa para aceitar"))
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// ExcluirConversa apaga a conversa (ou recusa um pedido) com uma pessoa
func ExcluirConversa(w http.ResponseWriter, r *http.Request) {
	eu, outro, ok := parametroDeMensagens(w, r, "usuarioId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if erro = repositorios.NovoRepositorioDeMensagens(db).ExcluirConversa(eu, outro); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// ExcluirMensagem apaga uma mensagem enviada pelo próprio usuário
func ExcluirMensagem(w http.ResponseWriter, r *http.Request) {
	eu, mensagemID, ok := parametroDeMensagens(w, r, "mensagemId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	apagada, erro := repositorios.NovoRepositorioDeMensagens(db).ExcluirMensagem(eu, mensagemID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if !apagada {
		respostas.Erro(w, http.StatusForbidden, errors.New("Você só pode apagar mensagens enviadas por você"))
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
