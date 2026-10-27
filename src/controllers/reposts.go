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
	"strings"
	"unicode/utf8"

	"github.com/gorilla/mux"
)

// Repostar compartilha uma publicação no perfil de quem está logado, com um comentário opcional (citação)
func Repostar(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}
	publicacaoID, erro := strconv.ParseUint(mux.Vars(r)["publicacaoId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	var corpo struct {
		Conteudo string `json:"conteudo"`
	}
	json.NewDecoder(r.Body).Decode(&corpo)
	comentario := strings.TrimSpace(corpo.Conteudo)
	if utf8.RuneCountInString(comentario) > 5000 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("O comentário deve ter no máximo 5000 caracteres"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if !verificarAcessoPublicacao(w, db, publicacaoID, usuarioID) {
		return
	}

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	// repostar um repost leva à publicação de origem
	originalID, erro := repositorio.OriginalDoRepost(publicacaoID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	pode, erro := repositorio.PodeRepostar(originalID, usuarioID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if !pode {
		respostas.Erro(w, http.StatusForbidden, errors.New("Publicações de contas privadas não podem ser repostadas"))
		return
	}

	if comentario == "" {
		existente, erro := repositorio.RepostSimples(usuarioID, originalID)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		if existente != 0 {
			respostas.Erro(w, http.StatusConflict, errors.New("Você já repostou esta publicação"))
			return
		}
	}

	id, erro := repositorio.Repostar(usuarioID, originalID, comentario)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	registrarNotificacao(db, func(n *repositorios.Notificacoes) error {
		if erro := n.NotificarRepost(originalID, usuarioID); erro != nil {
			return erro
		}
		return n.NotificarMencoes(comentario, id, 0, usuarioID)
	})
	respostas.JSON(w, http.StatusCreated, map[string]uint64{"id": id})
}

// DesfazerRepost remove o repost sem comentário que o usuário fez de uma publicação
func DesfazerRepost(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}
	publicacaoID, erro := strconv.ParseUint(mux.Vars(r)["publicacaoId"], 10, 64)
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

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	originalID, erro := repositorio.OriginalDoRepost(publicacaoID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	repostID, erro := repositorio.RepostSimples(usuarioID, originalID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if repostID == 0 {
		respostas.Erro(w, http.StatusNotFound, errors.New("Repost não encontrado"))
		return
	}
	if erro = repositorio.Deletar(repostID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
