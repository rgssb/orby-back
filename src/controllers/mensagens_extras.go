package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/eventos"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"time"
)

// LimiteImagemMensagem é o tamanho máximo da imagem de uma mensagem
const LimiteImagemMensagem = 4 << 20

func repositorioDeMensagens(w http.ResponseWriter) (*repositorios.Mensagens, func(), bool) {
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return nil, nil, false
	}
	repositorio := repositorios.NovoRepositorioDeMensagens(db)
	return repositorio, func() { db.Close() }, true
}

// ReagirAMensagem alterna uma reação com emoji em uma mensagem
func ReagirAMensagem(w http.ResponseWriter, r *http.Request) {
	eu, mensagemID, ok := parametroDeMensagens(w, r, "mensagemId")
	if !ok {
		return
	}
	var corpo struct {
		Emoji string `json:"emoji"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil || !modelos.EmojisPermitidos[corpo.Emoji] {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Reação inválida"))
		return
	}
	repositorio, fechar, ok := repositorioDeMensagens(w)
	if !ok {
		return
	}
	defer fechar()

	if erro := repositorio.Reagir(eu, mensagemID, corpo.Emoji); erro != nil {
		erroDeMensagem(w, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// EnviarImagemDeMensagem anexa uma imagem (corpo bruto) a uma mensagem já enviada
func EnviarImagemDeMensagem(w http.ResponseWriter, r *http.Request) {
	eu, mensagemID, ok := parametroDeMensagens(w, r, "mensagemId")
	if !ok {
		return
	}
	dados, erro := io.ReadAll(http.MaxBytesReader(w, r.Body, LimiteImagemMensagem))
	if erro != nil {
		respostas.Erro(w, http.StatusRequestEntityTooLarge, errors.New("A imagem deve ter no máximo 4 MB"))
		return
	}
	tipo := http.DetectContentType(dados)
	if len(dados) == 0 || !tiposDeFotoPermitidos[tipo] {
		respostas.Erro(w, http.StatusUnsupportedMediaType, errors.New("Envie uma imagem JPG, PNG, WEBP ou GIF"))
		return
	}
	repositorio, fechar, ok := repositorioDeMensagens(w)
	if !ok {
		return
	}
	defer fechar()

	if erro = repositorio.SalvarImagem(eu, mensagemID, tipo, dados); erro != nil {
		erroDeMensagem(w, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// BuscarImagemDeMensagem devolve a imagem de uma mensagem para os participantes da conversa
func BuscarImagemDeMensagem(w http.ResponseWriter, r *http.Request) {
	eu, mensagemID, ok := parametroDeMensagens(w, r, "mensagemId")
	if !ok {
		return
	}
	repositorio, fechar, ok := repositorioDeMensagens(w)
	if !ok {
		return
	}
	defer fechar()

	tipo, dados, erro := repositorio.BuscarImagem(eu, mensagemID)
	if erro != nil {
		erroDeMensagem(w, erro)
		return
	}
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(dados)
}

func alterarSilencio(w http.ResponseWriter, r *http.Request, silenciar bool) {
	eu, outro, ok := parametroDeMensagens(w, r, "usuarioId")
	if !ok {
		return
	}
	repositorio, fechar, ok := repositorioDeMensagens(w)
	if !ok {
		return
	}
	defer fechar()

	if erro := repositorio.Silenciar(eu, outro, silenciar); erro != nil {
		erroDeMensagem(w, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// SilenciarConversa deixa de contar as mensagens da conversa nas notificações
func SilenciarConversa(w http.ResponseWriter, r *http.Request) { alterarSilencio(w, r, true) }

// DessilenciarConversa volta a contar as mensagens da conversa
func DessilenciarConversa(w http.ResponseWriter, r *http.Request) { alterarSilencio(w, r, false) }

// AvisarDigitacao informa à outra pessoa que o usuário está digitando
func AvisarDigitacao(w http.ResponseWriter, r *http.Request) {
	eu, outro, ok := parametroDeMensagens(w, r, "usuarioId")
	if !ok {
		return
	}
	eventos.Publicar(outro, eventos.Evento{Tipo: "digitando", De: eu})
	respostas.JSON(w, http.StatusNoContent, nil)
}

// Eventos mantém a conexão aberta e envia avisos em tempo real (Server-Sent Events)
func Eventos(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroDeMensagens(w, r, "")
	if !ok {
		return
	}
	enviador, ok := w.(http.Flusher)
	if !ok {
		respostas.Erro(w, http.StatusInternalServerError, errors.New("Streaming não suportado"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	canal, cancelar := eventos.Assinar(eu)
	defer cancelar()
	fmt.Fprint(w, ": conectado\n\n")
	enviador.Flush()

	batida := time.NewTicker(25 * time.Second)
	defer batida.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-batida.C:
			fmt.Fprint(w, ": ping\n\n")
		case evento := <-canal:
			dados, _ := json.Marshal(evento)
			fmt.Fprintf(w, "data: %s\n\n", dados)
		}
		enviador.Flush()
	}
}
