package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"

	"github.com/gorilla/mux"
)

func parametroAutenticado(w http.ResponseWriter, r *http.Request, nome string) (eu, id uint64, ok bool) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return 0, 0, false
	}
	if nome == "" {
		return eu, 0, true
	}
	id, erro = strconv.ParseUint(mux.Vars(r)[nome], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Identificador inválido"))
		return 0, 0, false
	}
	return eu, id, true
}

func alterarSalvo(w http.ResponseWriter, r *http.Request, salvar bool) {
	eu, publicacaoID, ok := parametroAutenticado(w, r, "publicacaoId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	if salvar {
		if !verificarAcessoPublicacao(w, db, publicacaoID, eu) {
			return
		}
		erro = repositorio.Salvar(eu, publicacaoID)
	} else {
		erro = repositorio.RemoverSalvo(eu, publicacaoID)
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// SalvarPublicacao guarda a publicação nos salvos do usuário
func SalvarPublicacao(w http.ResponseWriter, r *http.Request) { alterarSalvo(w, r, true) }

// RemoverPublicacaoSalva tira a publicação dos salvos do usuário
func RemoverPublicacaoSalva(w http.ResponseWriter, r *http.Request) { alterarSalvo(w, r, false) }

// BuscarPublicacoesSalvas lista as publicações salvas do usuário (limite e deslocamento paginam)
func BuscarPublicacoesSalvas(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	limite, _ := strconv.ParseUint(r.URL.Query().Get("limite"), 10, 64)
	if limite == 0 || limite > 50 {
		limite = 50
	}
	deslocamento, _ := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	publicacoes, erro := repositorio.BuscarSalvos(eu, limite, deslocamento)
	if erro == nil {
		erro = repositorio.AnexarRepostagens(publicacoes, eu)
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, publicacoes)
}

// DefinirPrivacidade torna o perfil do usuário privado ou público
func DefinirPrivacidade(w http.ResponseWriter, r *http.Request) {
	eu, usuarioID, ok := parametroAutenticado(w, r, "usuarioId")
	if !ok {
		return
	}
	if eu != usuarioID {
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possível alterar a privacidade de outro usuário"))
		return
	}
	var corpo struct {
		Privado bool `json:"privado"`
	}
	if erro := json.NewDecoder(r.Body).Decode(&corpo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if erro = repositorios.NovoRepositorioDeUsuarios(db).DefinirPrivacidade(eu, corpo.Privado); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

// BuscarPedidosParaSeguir lista quem pediu para seguir o usuário autenticado
func BuscarPedidosParaSeguir(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	pedidos, erro := repositorios.NovoRepositorioDeUsuarios(db).BuscarPedidos(eu)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if pedidos == nil {
		pedidos = []modelos.Usuario{}
	}
	respostas.JSON(w, http.StatusOK, pedidos)
}

func responderPedido(w http.ResponseWriter, r *http.Request, aceitar bool) {
	eu, solicitanteID, ok := parametroAutenticado(w, r, "usuarioId")
	if !ok {
		return
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDeUsuarios(db)
	if aceitar {
		aceito, erro := repositorio.AceitarPedido(eu, solicitanteID)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		if !aceito {
			respostas.Erro(w, http.StatusNotFound, errors.New("Pedido não encontrado"))
			return
		}
	} else if erro = repositorio.CancelarPedido(eu, solicitanteID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	registrarNotificacao(db, func(n *repositorios.Notificacoes) error {
		if erro := n.RemoverPedidoParaSeguir(eu, solicitanteID); erro != nil {
			return erro
		}
		if aceitar {
			return n.NotificarSeguidor(eu, solicitanteID)
		}
		return nil
	})
	respostas.JSON(w, http.StatusNoContent, nil)
}

// AceitarPedidoParaSeguir deixa a pessoa passar a seguir o usuário autenticado
func AceitarPedidoParaSeguir(w http.ResponseWriter, r *http.Request) { responderPedido(w, r, true) }

// RecusarPedidoParaSeguir descarta o pedido de seguimento
func RecusarPedidoParaSeguir(w http.ResponseWriter, r *http.Request) { responderPedido(w, r, false) }
