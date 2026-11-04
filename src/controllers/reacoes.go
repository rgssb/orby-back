package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
)

func ReacoesPublicacao(w http.ResponseWriter, r *http.Request) {
	tratarReacoes(w, r, false)
}

func ReacoesComentario(w http.ResponseWriter, r *http.Request) {
	tratarReacoes(w, r, true)
}

func tratarReacoes(w http.ResponseWriter, r *http.Request, comentario bool) {
	parametro := "publicacaoId"
	if comentario {
		parametro = "comentarioId"
	}
	usuario, id, ok := parametroAutenticado(w, r, parametro)
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	repositorio := repositorios.NovoRepositorioDeReacoes(db)
	if r.Method == http.MethodGet {
		lista, erro := repositorio.Listar(id, usuario, comentario)
		if erro != nil {
			erroDeReacao(w, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, lista)
		return
	}
	var corpo struct {
		Emoji string `json:"emoji"`
		Ativa *bool  `json:"ativa"`
	}
	if erro = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&corpo); erro != nil || !modelos.EmojisPermitidos[corpo.Emoji] || corpo.Ativa == nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Reação inválida"))
		return
	}
	if erro = repositorio.Definir(id, usuario, comentario, corpo.Emoji, *corpo.Ativa); erro != nil {
		erroDeReacao(w, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

func erroDeReacao(w http.ResponseWriter, erro error) {
	if errors.Is(erro, repositorios.ErrReacaoIndisponivel) {
		respostas.Erro(w, http.StatusNotFound, erro)
		return
	}
	respostas.Erro(w, http.StatusInternalServerError, erro)
}
