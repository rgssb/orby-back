package controllers

import (
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// LimiteFoto é o tamanho máximo aceito para uma foto de perfil
const LimiteFoto = 2 << 20

var tiposDeFotoPermitidos = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// BuscarFoto devolve a foto de perfil de um usuario
func BuscarFoto(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
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

	tipo, dados, erro := repositorios.NovoRepositorioDeUsuarios(db).BuscarFoto(usuarioID)
	if erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Este usuário não tem foto"))
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	w.Header().Set("Content-Type", tipo)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	w.Write(dados)
}

// SalvarFoto grava a foto de perfil enviada (corpo bruto da requisicao) do usuario autenticado
func SalvarFoto(w http.ResponseWriter, r *http.Request) {
	usuarioID, ok := usuarioDaRotaIgualAoToken(w, r)
	if !ok {
		return
	}

	dados, erro := io.ReadAll(http.MaxBytesReader(w, r.Body, LimiteFoto))
	if erro != nil {
		respostas.Erro(w, http.StatusRequestEntityTooLarge, errors.New("A foto deve ter no máximo 2 MB"))
		return
	}

	if len(dados) == 0 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Nenhuma imagem foi enviada"))
		return
	}

	tipo := http.DetectContentType(dados)
	if !tiposDeFotoPermitidos[tipo] {
		respostas.Erro(w, http.StatusUnsupportedMediaType, errors.New("Envie uma imagem JPG, PNG, WEBP ou GIF"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if erro = repositorios.NovoRepositorioDeUsuarios(db).SalvarFoto(usuarioID, tipo, dados); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// RemoverFoto apaga a foto de perfil do usuario autenticado
func RemoverFoto(w http.ResponseWriter, r *http.Request) {
	usuarioID, ok := usuarioDaRotaIgualAoToken(w, r)
	if !ok {
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if erro = repositorios.NovoRepositorioDeUsuarios(db).RemoverFoto(usuarioID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// presetsDeBanner são os temas prontos que o usuário pode escolher
var presetsDeBanner = map[string]bool{"aurora": true, "por-do-sol": true, "oceano": true, "floresta": true}

// BuscarBanner devolve a imagem própria do banner de um usuario
func BuscarBanner(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
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

	tipo, dados, erro := repositorios.NovoRepositorioDeUsuarios(db).BuscarBannerImagem(usuarioID)
	if erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Este usuário não tem banner próprio"))
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	w.Header().Set("Content-Type", tipo)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	w.Write(dados)
}

// SalvarBanner grava a imagem do banner (corpo bruto) do usuario autenticado
func SalvarBanner(w http.ResponseWriter, r *http.Request) {
	usuarioID, ok := usuarioDaRotaIgualAoToken(w, r)
	if !ok {
		return
	}

	dados, erro := io.ReadAll(http.MaxBytesReader(w, r.Body, LimiteFoto))
	if erro != nil {
		respostas.Erro(w, http.StatusRequestEntityTooLarge, errors.New("O banner deve ter no máximo 2 MB"))
		return
	}

	if len(dados) == 0 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Nenhuma imagem foi enviada"))
		return
	}

	tipo := http.DetectContentType(dados)
	if !tiposDeFotoPermitidos[tipo] {
		respostas.Erro(w, http.StatusUnsupportedMediaType, errors.New("Envie uma imagem JPG, PNG, WEBP ou GIF"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if erro = repositorios.NovoRepositorioDeUsuarios(db).SalvarBannerImagem(usuarioID, tipo, dados); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// DefinirBannerPreset escolhe um dos temas prontos para o banner do usuario autenticado
func DefinirBannerPreset(w http.ResponseWriter, r *http.Request) {
	usuarioID, ok := usuarioDaRotaIgualAoToken(w, r)
	if !ok {
		return
	}

	var corpo struct {
		Preset string `json:"preset"`
	}
	if erro := json.NewDecoder(io.LimitReader(r.Body, 1<<10)).Decode(&corpo); erro != nil || !presetsDeBanner[corpo.Preset] {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Tema de banner inválido"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if erro = repositorios.NovoRepositorioDeUsuarios(db).SalvarBannerPreset(usuarioID, corpo.Preset); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// RemoverBanner volta o banner do usuario autenticado ao padrão
func RemoverBanner(w http.ResponseWriter, r *http.Request) {
	usuarioID, ok := usuarioDaRotaIgualAoToken(w, r)
	if !ok {
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if erro = repositorios.NovoRepositorioDeUsuarios(db).RemoverBanner(usuarioID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// usuarioDaRotaIgualAoToken garante que o usuario da URL e o dono do token; responde o erro quando nao for
func usuarioDaRotaIgualAoToken(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	usuarioIDNoToken, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return 0, false
	}

	usuarioID, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return 0, false
	}

	if usuarioID != usuarioIDNoToken {
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possível alterar algo que não é seu"))
		return 0, false
	}

	return usuarioID, true
}
