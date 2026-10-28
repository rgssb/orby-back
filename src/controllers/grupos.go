package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func erroDeGrupo(w http.ResponseWriter, erro error) {
	switch {
	case errors.Is(erro, repositorios.ErrGrupoInexistente), errors.Is(erro, repositorios.ErrUsuarioInexistente):
		respostas.Erro(w, http.StatusNotFound, erro)
	case errors.Is(erro, repositorios.ErrGrupoSemMembros):
		respostas.Erro(w, http.StatusBadRequest, erro)
	case errors.Is(erro, repositorios.ErrSemPermissao), errors.Is(erro, repositorios.ErrBloqueado), errors.Is(erro, repositorios.ErrPrivacidadeInteracao):
		respostas.Erro(w, http.StatusForbidden, erro)
	case errors.Is(erro, repositorios.ErrGrupoCheio), errors.Is(erro, repositorios.ErrJaEMembro):
		respostas.Erro(w, http.StatusConflict, erro)
	default:
		respostas.Erro(w, http.StatusInternalServerError, erro)
	}
}

// comGrupos abre o banco e entrega o repositório, o usuário e o id do grupo (quando a rota tem {grupoId})
func comGrupos(w http.ResponseWriter, r *http.Request, f func(repo *repositorios.Grupos, eu, grupoID uint64)) {
	eu, _, ok := parametroDeMensagens(w, r, "")
	if !ok {
		return
	}
	var grupoID uint64
	if bruto, tem := mux.Vars(r)["grupoId"]; tem {
		var erro error
		if grupoID, erro = strconv.ParseUint(bruto, 10, 64); erro != nil {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
			return
		}
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	f(repositorios.NovoRepositorioDeGrupos(db), eu, grupoID)
}

func usuarioDaRota(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, erro := strconv.ParseUint(mux.Vars(r)["usuarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return 0, false
	}
	return id, true
}

// CriarGrupo cria um grupo de conversa com as pessoas escolhidas
func CriarGrupo(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Nome    string   `json:"nome"`
		Membros []uint64 `json:"membros"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}
	nome, erro := modelos.PrepararNomeDoGrupo(corpo.Nome)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, _ uint64) {
		id, erro := repo.Criar(eu, nome, corpo.Membros)
		if erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusCreated, map[string]uint64{"id": id})
	})
}

// ListarGrupos devolve os grupos do usuário autenticado
func ListarGrupos(w http.ResponseWriter, r *http.Request) {
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, _ uint64) {
		grupos, erro := repo.Listar(eu)
		if erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, grupos)
	})
}

// BuscarGrupo abre um grupo (use ?apos=ID para trazer só as mensagens novas)
func BuscarGrupo(w http.ResponseWriter, r *http.Request) {
	apos, _ := strconv.ParseUint(r.URL.Query().Get("apos"), 10, 64)
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		dialogo, erro := repo.Abrir(eu, grupoID, apos)
		if erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, dialogo)
	})
}

// EnviarMensagemDeGrupo envia uma mensagem ao grupo
func EnviarMensagemDeGrupo(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Texto string `json:"texto"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}
	texto, erro := modelos.PrepararTexto(corpo.Texto)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		chave := strconv.FormatUint(eu, 10)
		if limitadorDeMensagens.Bloqueado(chave) {
			respostas.Erro(w, http.StatusTooManyRequests, errors.New("Você está enviando mensagens rápido demais. Aguarde um instante"))
			return
		}
		mensagem, erro := repo.Enviar(eu, grupoID, strings.TrimSpace(texto))
		if erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		limitadorDeMensagens.Registrar(chave)
		respostas.JSON(w, http.StatusCreated, mensagem)
	})
}

// AdicionarAoGrupo inclui uma pessoa no grupo
func AdicionarAoGrupo(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		UsuarioID uint64 `json:"usuarioId"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil || corpo.UsuarioID == 0 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		if erro := repo.Adicionar(eu, grupoID, corpo.UsuarioID); erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusNoContent, nil)
	})
}

// DefinirPermissaoNoGrupo deixa (ou impede) um membro de adicionar pessoas; só o criador pode
func DefinirPermissaoNoGrupo(w http.ResponseWriter, r *http.Request) {
	var corpo struct {
		Pode bool `json:"pode"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}
	alvo, ok := usuarioDaRota(w, r)
	if !ok {
		return
	}
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		if erro := repo.DefinirPermissao(eu, grupoID, alvo, corpo.Pode); erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusNoContent, nil)
	})
}

// RemoverDoGrupo tira um membro do grupo (ou o próprio usuário sai)
func RemoverDoGrupo(w http.ResponseWriter, r *http.Request) {
	alvo, ok := usuarioDaRota(w, r)
	if !ok {
		return
	}
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		if erro := repo.Remover(eu, grupoID, alvo); erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusNoContent, nil)
	})
}

// LimiteFotoDoGrupo é o tamanho máximo da foto de um grupo
const LimiteFotoDoGrupo = 2 << 20

// ListarCandidatosDeGrupo devolve as pessoas que podem entrar num grupo (?q= filtra por nome ou nick)
func ListarCandidatosDeGrupo(w http.ResponseWriter, r *http.Request) {
	busca := strings.Map(func(c rune) rune {
		if c == '%' || c == '_' || c == '\\' {
			return -1
		}
		return c
	}, strings.TrimSpace(strings.TrimPrefix(r.URL.Query().Get("q"), "@")))
	if len([]rune(busca)) > 50 {
		busca = string([]rune(busca)[:50])
	}
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, _ uint64) {
		pessoas, erro := repo.Candidatos(eu, busca)
		if erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusOK, pessoas)
	})
}

// EnviarFotoDoGrupo troca a foto do grupo (corpo bruto da imagem)
func EnviarFotoDoGrupo(w http.ResponseWriter, r *http.Request) {
	dados, erro := io.ReadAll(http.MaxBytesReader(w, r.Body, LimiteFotoDoGrupo))
	if erro != nil {
		respostas.Erro(w, http.StatusRequestEntityTooLarge, errors.New("A foto deve ter no máximo 2 MB"))
		return
	}
	tipo := http.DetectContentType(dados)
	if len(dados) == 0 || !tiposDeFotoPermitidos[tipo] {
		respostas.Erro(w, http.StatusUnsupportedMediaType, errors.New("Envie uma imagem JPG, PNG, WEBP ou GIF"))
		return
	}
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		if erro := repo.SalvarFoto(eu, grupoID, tipo, dados); erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusNoContent, nil)
	})
}

// BuscarFotoDoGrupo devolve a foto do grupo para os membros
func BuscarFotoDoGrupo(w http.ResponseWriter, r *http.Request) {
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		tipo, dados, erro := repo.BuscarFoto(eu, grupoID)
		if errors.Is(erro, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		w.Header().Set("Content-Type", tipo)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(dados)
	})
}

// RemoverFotoDoGrupo tira a foto do grupo
func RemoverFotoDoGrupo(w http.ResponseWriter, r *http.Request) {
	comGrupos(w, r, func(repo *repositorios.Grupos, eu, grupoID uint64) {
		if erro := repo.RemoverFoto(eu, grupoID); erro != nil {
			erroDeGrupo(w, erro)
			return
		}
		respostas.JSON(w, http.StatusNoContent, nil)
	})
}
