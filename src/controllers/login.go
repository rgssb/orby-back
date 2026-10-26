package controllers

import (
	"API/src/autenticacao"
	"API/src/banco"
	"API/src/modelos"
	"API/src/repositorios"
	"API/src/respostas"
	"API/src/seguranca"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

func Login(w http.ResponseWriter, r *http.Request) {
	corpoRequisicao, erro := ioutil.ReadAll(r.Body)
	if erro != nil {
		respostas.Erro(w, http.StatusUnprocessableEntity, erro)
		return
	}

	var usuario modelos.Usuario
	if erro = json.Unmarshal(corpoRequisicao, &usuario); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	chaveDoLimitador := enderecoDoCliente(r) + "|" + strings.ToLower(strings.TrimSpace(usuario.Email))
	if limitadorDeLogin.Bloqueado(chaveDoLimitador) {
		respostas.Erro(w, http.StatusTooManyRequests, errors.New("Muitas tentativas de login. Aguarde alguns minutos e tente novamente"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDeUsuarios(db)
	usuarioSalvoNoBanco, erro := repositorio.BuscarPorEmail(usuario.Email)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	if erro = seguranca.VerificarSenha(usuarioSalvoNoBanco.Senha, usuario.Senha); erro != nil {
		limitadorDeLogin.Registrar(chaveDoLimitador)
		respostas.Erro(w, http.StatusUnauthorized, errors.New("E-mail ou senha incorretos"))
		return
	}
	limitadorDeLogin.Reiniciar(chaveDoLimitador)

	token, erro := autenticacao.CriarToken(usuarioSalvoNoBanco.ID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	usuarioID := strconv.FormatUint(usuarioSalvoNoBanco.ID, 10)

	respostas.JSON(w, http.StatusOK, modelos.DadosAutenticacao{usuarioID, token})
}
