package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
)

func PreferenciasDeNotificacao(w http.ResponseWriter, r *http.Request) {
	usuario, erro := autenticacao.ExtrairUsuarioID(r)
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
	repo := repositorios.NovoRepositorioDeNotificacoes(db)
	if r.Method == http.MethodGet {
		tipos, erro := repo.TiposSilenciados(usuario)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, map[string][]string{"silenciados": tipos})
		return
	}
	var corpo struct {
		Silenciados []string `json:"silenciados"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if erro = json.NewDecoder(r.Body).Decode(&corpo); erro != nil || corpo.Silenciados == nil {
		respostas.Erro(w, http.StatusBadRequest, repositorios.ErrPreferenciasInvalidas)
		return
	}
	if erro = repo.SalvarTiposSilenciados(usuario, corpo.Silenciados); errors.Is(erro, repositorios.ErrPreferenciasInvalidas) {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
