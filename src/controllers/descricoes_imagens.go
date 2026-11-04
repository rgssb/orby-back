package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/respostas"
	"strings"
	"unicode/utf8"
)

// AtualizarDescricaoDaImagem altera a descrição de uma imagem do próprio usuário.
func AtualizarDescricaoDaImagem(w http.ResponseWriter, r *http.Request) {
	posicao, erro := posicaoDaImagem(r)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Endereço inválido"))
		return
	}
	var corpo struct {
		Descricao string `json:"descricao"`
	}
	if erro = json.NewDecoder(r.Body).Decode(&corpo); erro != nil || utf8.RuneCountInString(corpo.Descricao) > 500 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("A descrição da imagem deve ter no máximo 500 caracteres"))
		return
	}
	db, repositorio, publicacaoID, ok := publicacaoDoUsuarioAutenticado(w, r)
	if !ok {
		return
	}
	defer db.Close()
	if erro = repositorio.AtualizarDescricaoImagem(publicacaoID, posicao, strings.TrimSpace(corpo.Descricao)); erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Imagem não encontrada"))
		} else {
			respostas.Erro(w, http.StatusInternalServerError, erro)
		}
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
