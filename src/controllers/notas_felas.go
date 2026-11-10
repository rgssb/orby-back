package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"
	"strings"
	"unicode/utf8"
)

func NotasFelas(w http.ResponseWriter, r *http.Request) {
	eu, perfil, ok := parametroAutenticado(w, r, "usuarioId")
	if !ok {
		return
	}
	db, err := banco.Conectar()
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Close()
	if r.Method == http.MethodPost {
		var corpo struct {
			Conteudo string `json:"conteudo"`
		}
		if err = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&corpo); err != nil {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Nota inválida"))
			return
		}
		corpo.Conteudo = strings.TrimSpace(corpo.Conteudo)
		if n := utf8.RuneCountInString(corpo.Conteudo); n == 0 || n > 280 {
			respostas.Erro(w, http.StatusBadRequest, errors.New("A nota deve ter entre 1 e 280 caracteres"))
			return
		}
		err = repositorios.CriarNotaFela(db, perfil, eu, corpo.Conteudo)
		if tratarErroNotas(w, err) {
			return
		}
		respostas.JSON(w, http.StatusNoContent, nil)
		return
	}
	var antes uint64
	if valor := r.URL.Query().Get("antes"); valor != "" {
		antes, err = strconv.ParseUint(valor, 10, 64)
		if err != nil || antes == 0 {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Página inválida"))
			return
		}
	}
	resultado, err := repositorios.ListarNotasFelas(db, perfil, eu, antes)
	if tratarErroNotas(w, err) {
		return
	}
	respostas.JSON(w, http.StatusOK, resultado)
}

func ExcluirNotaFela(w http.ResponseWriter, r *http.Request) {
	eu, id, ok := parametroAutenticado(w, r, "notaId")
	if !ok {
		return
	}
	db, err := banco.Conectar()
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Close()
	if tratarErroNotas(w, repositorios.ExcluirNotaFela(db, id, eu)) {
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

func PreferenciaNotasFelas(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	db, err := banco.Conectar()
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Close()
	if r.Method == http.MethodGet {
		var ativas bool
		err = db.QueryRow("select notas_felas_ativas from usuarios where id = ?", eu).Scan(&ativas)
		if tratarErroNotas(w, err) {
			return
		}
		respostas.JSON(w, http.StatusOK, map[string]bool{"ativas": ativas})
		return
	}
	var corpo struct {
		Ativas *bool `json:"ativas"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&corpo); err != nil || corpo.Ativas == nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Preferência inválida"))
		return
	}
	_, err = db.Exec("update usuarios set notas_felas_ativas = ? where id = ?", *corpo.Ativas, eu)
	if tratarErroNotas(w, err) {
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

func tratarErroNotas(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, repositorios.ErrNotasIndisponiveis) {
		respostas.Erro(w, http.StatusNotFound, errors.New("Notas indisponíveis"))
	} else {
		respostas.Erro(w, http.StatusInternalServerError, err)
	}
	return true
}
