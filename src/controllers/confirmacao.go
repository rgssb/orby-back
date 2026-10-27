package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"orby/api/src/banco"
	"orby/api/src/config"
	"orby/api/src/email"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"orby/api/src/seguranca"
	"strings"
	"time"
)

var (
	limitadorDeReenvio         = seguranca.NovoLimitador(5, 15*time.Minute)
	limitadorDeConfirmacao     = seguranca.NovoLimitador(10, 15*time.Minute)
	mensagemConfirmacaoEnviada = "Se houver uma conta aguardando confirmação para esse e-mail, enviamos um novo link."
)

// enviarConfirmacaoDeEmail cria um link novo e o envia sem travar a resposta; erros só vão para o log
func enviarConfirmacaoDeEmail(repositorio interface {
	CriarConfirmacaoDeEmail(uint64, string) (bool, error)
}, usuarioID uint64, nome, destinatario string) {
	bytesDoToken := make([]byte, 32)
	if _, erro := rand.Read(bytesDoToken); erro != nil {
		log.Printf("erro ao gerar token de confirmação: %v", erro)
		return
	}
	token := hex.EncodeToString(bytesDoToken)

	criado, erro := repositorio.CriarConfirmacaoDeEmail(usuarioID, hashDoToken(token))
	if erro != nil {
		log.Printf("erro ao guardar token de confirmação: %v", erro)
		return
	}
	if !criado {
		return
	}

	link := config.FrontURL + "/confirmar-email?token=" + url.QueryEscape(token)
	go func() {
		if erro := email.EnviarConfirmacaoDeEmail(destinatario, nome, link); erro != nil {
			log.Printf("erro ao enviar e-mail de confirmação: %v", erro)
		}
	}()
}

// ConfirmarEmail ativa a conta a partir do token recebido por e-mail
func ConfirmarEmail(w http.ResponseWriter, r *http.Request) {
	cliente := enderecoDoCliente(r)
	if limitadorDeConfirmacao.Bloqueado(cliente) {
		respostas.Erro(w, http.StatusTooManyRequests, errors.New("Muitas tentativas. Tente novamente em alguns minutos"))
		return
	}
	limitadorDeConfirmacao.Registrar(cliente)

	var corpo struct {
		Token string `json:"token"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil || corpo.Token == "" {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Link de confirmação inválido"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	erro = repositorios.NovoRepositorioDeUsuarios(db).ConfirmarEmailComToken(hashDoToken(corpo.Token))
	if errors.Is(erro, repositorios.ErroLinkDeConfirmacaoInvalido) {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// ReenviarConfirmacao manda um novo link. A resposta é sempre a mesma, exista ou não a conta,
// para não revelar quem está cadastrado.
func ReenviarConfirmacao(w http.ResponseWriter, r *http.Request) {
	cliente := enderecoDoCliente(r)
	if limitadorDeReenvio.Bloqueado(cliente) {
		respostas.Erro(w, http.StatusTooManyRequests, errors.New("Muitas tentativas. Tente novamente em alguns minutos"))
		return
	}
	limitadorDeReenvio.Registrar(cliente)

	var corpo struct {
		Email string `json:"email"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil || strings.TrimSpace(corpo.Email) == "" {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Informe o e-mail da sua conta"))
		return
	}
	emailInformado := strings.TrimSpace(corpo.Email)

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDeUsuarios(db)
	usuarioID, nome, confirmado, erro := repositorio.BuscarParaConfirmacao(emailInformado)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if usuarioID != 0 && !confirmado {
		enviarConfirmacaoDeEmail(repositorio, usuarioID, nome, emailInformado)
	}

	respostas.JSON(w, http.StatusOK, map[string]string{"mensagem": mensagemConfirmacaoEnviada})
}
