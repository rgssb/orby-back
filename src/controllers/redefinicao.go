package controllers

import (
	"orby/api/src/banco"
	"orby/api/src/config"
	"orby/api/src/email"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"orby/api/src/seguranca"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	limitadorDeLogin           = seguranca.NovoLimitador(8, 15*time.Minute)
	limitadorDeRecuperacao     = seguranca.NovoLimitador(5, 15*time.Minute)
	limitadorDeRedefinicao     = seguranca.NovoLimitador(10, 15*time.Minute)
	mensagemSolicitacaoEnviada = "Se o e-mail estiver cadastrado, enviamos um link para redefinir a senha."
)

func enderecoDoCliente(r *http.Request) string {
	host, _, erro := net.SplitHostPort(r.RemoteAddr)
	if erro != nil {
		return r.RemoteAddr
	}
	return host
}

func hashDoToken(token string) string {
	resumo := sha256.Sum256([]byte(token))
	return hex.EncodeToString(resumo[:])
}

// EsqueciSenha gera um link de recuperação e o envia por e-mail. A resposta é sempre a
// mesma, exista ou não o e-mail, para não revelar quem está cadastrado.
func EsqueciSenha(w http.ResponseWriter, r *http.Request) {
	cliente := enderecoDoCliente(r)
	if limitadorDeRecuperacao.Bloqueado(cliente) {
		respostas.Erro(w, http.StatusTooManyRequests, errors.New("Muitas tentativas. Tente novamente em alguns minutos"))
		return
	}
	limitadorDeRecuperacao.Registrar(cliente)

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
	usuarioID, nome, erro := repositorio.BuscarParaRecuperacao(emailInformado)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	if usuarioID != 0 {
		bytesDoToken := make([]byte, 32)
		if _, erro = rand.Read(bytesDoToken); erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		token := hex.EncodeToString(bytesDoToken)

		criado, erro := repositorio.CriarRedefinicaoDeSenha(usuarioID, hashDoToken(token))
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}

		if criado {
			link := config.FrontURL + "/redefinir-senha?token=" + url.QueryEscape(token)
			go func() {
				if erro := email.EnviarRecuperacaoDeSenha(emailInformado, nome, link); erro != nil {
					log.Printf("erro ao enviar e-mail de recuperação: %v", erro)
				}
			}()
		}
	}

	respostas.JSON(w, http.StatusOK, map[string]string{"mensagem": mensagemSolicitacaoEnviada})
}

// RedefinirSenha troca a senha usando o token recebido por e-mail
func RedefinirSenha(w http.ResponseWriter, r *http.Request) {
	cliente := enderecoDoCliente(r)
	if limitadorDeRedefinicao.Bloqueado(cliente) {
		respostas.Erro(w, http.StatusTooManyRequests, errors.New("Muitas tentativas. Tente novamente em alguns minutos"))
		return
	}
	limitadorDeRedefinicao.Registrar(cliente)

	var corpo struct {
		Token string `json:"token"`
		Nova  string `json:"nova"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}

	if erro := modelos.ValidarNovaSenha(corpo.Nova); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	senhaComHash, erro := seguranca.Hash(corpo.Nova)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	erro = repositorios.NovoRepositorioDeUsuarios(db).RedefinirSenhaComToken(hashDoToken(corpo.Token), string(senhaComHash))
	if errors.Is(erro, repositorios.ErroTokenInvalido) {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}
