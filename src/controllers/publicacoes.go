package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"

	"github.com/gorilla/mux"
)

// CriarPublicacao cria uma nova puvlicacao no banco de dados
func CriarPublicacao(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	corpoRequisicao, erro := ioutil.ReadAll(r.Body)
	if erro != nil {
		respostas.Erro(w, http.StatusUnprocessableEntity, erro)
		return
	}

	var publicacao modelos.Publicacao
	if erro := json.Unmarshal(corpoRequisicao, &publicacao); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	publicacao.AutorID = usuarioID

	if erro = publicacao.Preparar(); erro != nil {
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
	publicacaoID, erro := repositorio.Criar(publicacao)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	publicacao.ID = publicacaoID
	registrarNotificacao(db, func(n *repositorios.Notificacoes) error {
		return n.NotificarMencoes(publicacao.Conteudo, publicacaoID, 0, usuarioID)
	})
	if publicacao.Felas {
		erro = repositorio.AvisarNovaPublicacaoFelas(usuarioID)
	} else {
		erro = repositorio.AvisarNovaPublicacao(usuarioID)
	}
	if erro != nil {
		log.Printf("aviso de nova publicacao: %v", erro)
	}
	respostas.JSON(w, http.StatusCreated, publicacao)

}

// BuscarPublicacoes busca as publocacoes que apareceriam no feed do usuario
func BuscarPublicacoes(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
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

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	limite, _ := strconv.ParseUint(r.URL.Query().Get("limite"), 10, 64)
	if limite > 50 {
		limite = 50
	}
	deslocamento, _ := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)

	publicacoes, erro := repositorio.Buscar(usuarioID, limite, deslocamento, r.URL.Query().Get("aba"), r.URL.Query().Get("ordem"))
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	if publicacoes == nil {
		publicacoes = []modelos.Publicacao{}
	}
	if erro = repositorio.AnexarRepostagens(publicacoes, usuarioID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusOK, publicacoes)

}

// BuscarPublicacao traz uma unica publicacao
func BuscarPublicacao(w http.ResponseWriter, r *http.Request) {
	parametros := mux.Vars(r)
	publicacaoID, erro := strconv.ParseUint(parametros["publicacaoId"], 10, 64)
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

	visitanteID, _ := autenticacao.ExtrairUsuarioID(r)
	pode, moderacao := verificarLeituraPublicacao(w, db, publicacaoID, visitanteID, "publicacao")
	if !pode {
		return
	}

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	publicacao, erro := repositorio.BuscarPorID(publicacaoID)
	if erro != nil {
		if erro == sql.ErrNoRows {
			respostas.Erro(w, http.StatusNotFound, erro)
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	if usuarioID, erro := autenticacao.ExtrairUsuarioID(r); erro == nil {
		publicacao.CurtidoPorUsuario, _ = repositorio.CurtidaPor(publicacaoID, usuarioID)
	}
	anexadas := []modelos.Publicacao{publicacao}
	if erro = repositorio.AnexarRepostagens(anexadas, visitanteID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	publicacao = anexadas[0]
	publicacao.AcessoModeracao = moderacao

	respostas.JSON(w, http.StatusOK, publicacao)

}

// AtualizarPublicacao altera os dados de uma publicacao
func AtualizarPublicacao(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	parametros := mux.Vars(r)
	publicacaoID, erro := strconv.ParseUint(parametros["publicacaoId"], 10, 64)
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
	publicacaoSalvaNoBanco, erro := repositorio.BuscarPorID(publicacaoID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if publicacaoSalvaNoBanco.AutorID != usuarioID {
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possivel alterar algo que não seja seu"))
		return
	}

	corpoRequisicao, erro := ioutil.ReadAll(r.Body)
	if erro != nil {
		respostas.Erro(w, http.StatusUnprocessableEntity, erro)
		return
	}

	var publicacao modelos.Publicacao
	if erro = json.Unmarshal(corpoRequisicao, &publicacao); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	if erro = publicacao.Preparar(); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	if erro = repositorio.Atualizar(publicacaoID, publicacao); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// DeletarPublicacao exclui os dados de uma publicacao
func DeletarPublicacao(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	parametros := mux.Vars(r)
	publicacaoID, erro := strconv.ParseUint(parametros["publicacaoId"], 10, 64)
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
	publicacaoSalvaNoBanco, erro := repositorio.BuscarPorID(publicacaoID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	if publicacaoSalvaNoBanco.AutorID != usuarioID {
		admin, erro := repositorios.EhAdmin(db, usuarioID)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		if !admin {
			respostas.Erro(w, http.StatusForbidden, errors.New("Não é possivel apagar algo que não seja seu"))
			return
		}
		erro = repositorio.DeletarDeContaSuspensa(publicacaoID)
		switch {
		case errors.Is(erro, repositorios.ErrContaAtiva):
			respostas.Erro(w, http.StatusForbidden, erro)
			return
		case errors.Is(erro, sql.ErrNoRows):
			respostas.Erro(w, http.StatusNotFound, errors.New("Publicação não encontrada"))
			return
		case erro != nil:
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		log.Printf("administração: usuário %d apagou a publicação %d de %d", usuarioID, publicacaoID, publicacaoSalvaNoBanco.AutorID)
		respostas.JSON(w, http.StatusNoContent, nil)
		return
	}

	if erro = repositorio.Deletar(publicacaoID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// BuscarPublicacoesPorUsuario traz todas as publicacoes de um usuario especifico
func BuscarPublicacoesPorUsuario(w http.ResponseWriter, r *http.Request) {
	usuarioLogadoID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	parametros := mux.Vars(r)
	usuarioID, erro := strconv.ParseUint(parametros["usuarioId"], 10, 64)
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

	moderacao := false
	admin := false
	if usuarioID != usuarioLogadoID {
		admin, erro = repositorios.EhAdmin(db, usuarioLogadoID)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		privado, segue, _, erro := repositorios.NovoRepositorioDeUsuarios(db).Privacidade(usuarioID, usuarioLogadoID)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		if privado && !segue {
			moderacao = admin
			if !moderacao {
				respostas.JSON(w, http.StatusOK, []modelos.Publicacao{})
				return
			}
			if erro = repositorios.RegistrarAcessoModeracao(db, usuarioLogadoID, usuarioID, 0, "perfil_publicacoes"); erro != nil {
				respostas.Erro(w, http.StatusInternalServerError, erro)
				return
			}
		}
	}

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	var publicacoes []modelos.Publicacao
	if admin {
		publicacoes, erro = repositorio.BuscarPorUsuarioModeracao(usuarioID, usuarioLogadoID)
	} else {
		publicacoes, erro = repositorio.BuscarPorUsuario(usuarioID, usuarioLogadoID)
	}
	if erro == nil {
		erro = repositorio.AnexarRepostagens(publicacoes, usuarioLogadoID)
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	for i := range publicacoes {
		if admin {
			visivel, erro := repositorio.PodeVer(publicacoes[i].ID, usuarioLogadoID)
			if erro != nil {
				respostas.Erro(w, http.StatusInternalServerError, erro)
				return
			}
			if !visivel {
				if !moderacao {
					if erro = repositorios.RegistrarAcessoModeracao(db, usuarioLogadoID, usuarioID, publicacoes[i].ID, "perfil_publicacao"); erro != nil {
						respostas.Erro(w, http.StatusInternalServerError, erro)
						return
					}
				}
				publicacoes[i].AcessoModeracao = true
			}
		}
	}
	respostas.JSON(w, http.StatusOK, publicacoes)
}

// CurtirPublicacao adiciona uma curtida na publicacao
func CurtirPublicacao(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	parametros := mux.Vars(r)
	publicacaoID, erro := strconv.ParseUint(parametros["publicacaoId"], 10, 64)
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

	if !verificarAcessoPublicacao(w, db, publicacaoID, usuarioID) {
		return
	}

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	if erro = repositorio.Curtir(publicacaoID, usuarioID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	registrarNotificacao(db, func(n *repositorios.Notificacoes) error {
		return n.NotificarCurtidaDePublicacao(publicacaoID, usuarioID)
	})

	respostas.JSON(w, http.StatusNoContent, nil)
}

// DescurtirPublicacao retira uma curtida na publicaca
func DescurtirPublicacao(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	parametros := mux.Vars(r)
	publicacaoID, erro := strconv.ParseUint(parametros["publicacaoId"], 10, 64)
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
	if erro = repositorio.Descurtir(publicacaoID, usuarioID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	registrarNotificacao(db, func(n *repositorios.Notificacoes) error {
		return n.RemoverCurtidaDePublicacao(publicacaoID, usuarioID)
	})

	respostas.JSON(w, http.StatusNoContent, nil)
}
