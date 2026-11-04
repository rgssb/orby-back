package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strings"
	"unicode/utf8"
)

func erroPasta(w http.ResponseWriter, erro error) {
	if errors.Is(erro, repositorios.ErrPastaNaoEncontrada) {
		respostas.Erro(w, http.StatusNotFound, erro)
	} else {
		respostas.Erro(w, http.StatusInternalServerError, erro)
	}
}

func lerNomePasta(w http.ResponseWriter, r *http.Request) (string, bool) {
	var corpo struct {
		Nome string `json:"nome"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Nome de pasta inválido"))
		return "", false
	}
	nome := strings.TrimSpace(corpo.Nome)
	if nome == "" || utf8.RuneCountInString(nome) > 80 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Use um nome de 1 a 80 caracteres"))
		return "", false
	}
	return nome, true
}

func ListarPastas(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		erroPasta(w, erro)
		return
	}
	defer db.Close()
	pastas, erro := repositorios.NovoRepositorioDePublicacoes(db).ListarPastas(eu, 0)
	if erro != nil {
		erroPasta(w, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, pastas)
}

func CriarPasta(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	nome, ok := lerNomePasta(w, r)
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		erroPasta(w, erro)
		return
	}
	defer db.Close()
	id, erro := repositorios.NovoRepositorioDePublicacoes(db).CriarPasta(eu, nome)
	if erro != nil {
		erroPasta(w, erro)
		return
	}
	respostas.JSON(w, http.StatusCreated, repositorios.PastaSalvos{ID: id, Nome: nome})
}

func RenomearPasta(w http.ResponseWriter, r *http.Request) {
	eu, pastaID, ok := parametroAutenticado(w, r, "pastaId")
	if !ok {
		return
	}
	nome, ok := lerNomePasta(w, r)
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		erroPasta(w, erro)
		return
	}
	defer db.Close()
	if erro = repositorios.NovoRepositorioDePublicacoes(db).RenomearPasta(eu, pastaID, nome); erro != nil {
		erroPasta(w, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

func ExcluirPasta(w http.ResponseWriter, r *http.Request) {
	eu, pastaID, ok := parametroAutenticado(w, r, "pastaId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		erroPasta(w, erro)
		return
	}
	defer db.Close()
	if erro = repositorios.NovoRepositorioDePublicacoes(db).ExcluirPasta(eu, pastaID); erro != nil {
		erroPasta(w, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

func PastasDaPublicacao(w http.ResponseWriter, r *http.Request) {
	eu, publicacaoID, ok := parametroAutenticado(w, r, "publicacaoId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		erroPasta(w, erro)
		return
	}
	defer db.Close()
	if r.Method == http.MethodPost {
		nome, ok := lerNomePasta(w, r)
		if !ok {
			return
		}
		id, erro := repositorios.NovoRepositorioDePublicacoes(db).CriarPastaParaPublicacao(eu, publicacaoID, nome)
		if erro != nil {
			erroPasta(w, erro)
			return
		}
		respostas.JSON(w, http.StatusCreated, repositorios.PastaSalvos{ID: id, Nome: nome, Incluida: true})
		return
	}
	if r.Method == http.MethodGet {
		if !verificarAcessoPublicacao(w, db, publicacaoID, eu) {
			return
		}
		var salvo bool
		if erro = db.QueryRow("select exists(select 1 from salvos where usuario_id = ? and publicacao_id = ?)", eu, publicacaoID).Scan(&salvo); erro != nil {
			erroPasta(w, erro)
			return
		}
		if !salvo {
			erroPasta(w, repositorios.ErrPastaNaoEncontrada)
			return
		}
		pastas, erro := repositorios.NovoRepositorioDePublicacoes(db).ListarPastas(eu, publicacaoID)
		if erro != nil {
			erroPasta(w, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, pastas)
		return
	}
	var corpo struct {
		Pastas []uint64 `json:"pastas"`
	}
	if erro = json.NewDecoder(r.Body).Decode(&corpo); erro != nil || corpo.Pastas == nil || len(corpo.Pastas) > 200 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Lista de pastas inválida"))
		return
	}
	unicos := make(map[uint64]bool)
	for _, id := range corpo.Pastas {
		if id == 0 || unicos[id] {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Lista de pastas inválida"))
			return
		}
		unicos[id] = true
	}
	if erro = repositorios.NovoRepositorioDePublicacoes(db).DefinirPastas(eu, publicacaoID, corpo.Pastas); erro != nil {
		erroPasta(w, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
