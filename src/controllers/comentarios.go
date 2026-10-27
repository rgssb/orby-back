package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// BuscarComentarios lista os comentarios de uma publicacao
func BuscarComentarios(w http.ResponseWriter, r *http.Request) {
	publicacaoID, erro := strconv.ParseUint(mux.Vars(r)["publicacaoId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

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

	if !verificarAcessoPublicacao(w, db, publicacaoID, usuarioID) {
		return
	}

	limite, _ := strconv.ParseUint(r.URL.Query().Get("limite"), 10, 64)
	if limite > 500 {
		limite = 500
	}
	deslocamento, _ := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)

	comentarios, erro := repositorios.NovoRepositorioDeComentarios(db).BuscarPorPublicacao(publicacaoID, usuarioID, limite, deslocamento)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusOK, comentarios)
}

// BuscarRespostas lista as respostas de um comentario
func BuscarRespostas(w http.ResponseWriter, r *http.Request) {
	comentarioID, erro := strconv.ParseUint(mux.Vars(r)["comentarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

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

	repositorio := repositorios.NovoRepositorioDeComentarios(db)
	comentario, _, erro := repositorio.BuscarPorID(comentarioID)
	if errors.Is(erro, sql.ErrNoRows) {
		respostas.Erro(w, http.StatusNotFound, errors.New("Comentário não encontrado"))
		return
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	if !verificarAcessoPublicacao(w, db, comentario.PublicacaoID, usuarioID) {
		return
	}

	limite, _ := strconv.ParseUint(r.URL.Query().Get("limite"), 10, 64)
	if limite > 500 {
		limite = 500
	}
	deslocamento, _ := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)

	lista, erro := repositorio.BuscarRespostas(comentarioID, usuarioID, limite, deslocamento)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	respostas.JSON(w, http.StatusOK, lista)
}

// CriarComentario adiciona um comentario do usuario autenticado a uma publicacao
func CriarComentario(w http.ResponseWriter, r *http.Request) {
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

	var comentario modelos.Comentario
	if erro = json.NewDecoder(r.Body).Decode(&comentario); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}

	comentario.PublicacaoID = publicacaoID
	comentario.AutorID = usuarioID
	if erro = comentario.Preparar(); erro != nil {
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

	repositorioDeComentarios := repositorios.NovoRepositorioDeComentarios(db)
	var paiID, raizID uint64
	if comentario.ParentID != 0 {
		paiID, _, raizID, erro = repositorioDeComentarios.ResolverPai(publicacaoID, comentario.ParentID)
		if errors.Is(erro, repositorios.ErroComentarioPaiInvalido) {
			respostas.Erro(w, http.StatusBadRequest, erro)
			return
		}
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		comentario.ParentID = paiID
	}

	comentario.ID, erro = repositorioDeComentarios.Criar(comentario)
	if erro != nil {
		var erroMySQL *mysql.MySQLError
		if errors.As(erro, &erroMySQL) && erroMySQL.Number == 1452 {
			respostas.Erro(w, http.StatusNotFound, errors.New("Publicação não encontrada"))
			return
		}

		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	registrarNotificacao(db, func(n *repositorios.Notificacoes) error {
		if erro := n.NotificarComentario(publicacaoID, comentario.ID, usuarioID); erro != nil {
			return erro
		}
		if paiID != 0 {
			if erro := n.NotificarResposta(paiID, raizID, usuarioID); erro != nil {
				return erro
			}
		}
		return n.NotificarMencoes(comentario.Conteudo, publicacaoID, comentario.ID, usuarioID)
	})

	respostas.JSON(w, http.StatusCreated, comentario)
}

// DeletarComentario remove um comentario; so o autor do comentario ou o dono da publicacao podem fazer isso
func DeletarComentario(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	comentarioID, erro := strconv.ParseUint(mux.Vars(r)["comentarioId"], 10, 64)
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

	repositorio := repositorios.NovoRepositorioDeComentarios(db)
	comentario, donoDaPublicacao, erro := repositorio.BuscarPorID(comentarioID)
	if erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Comentário não encontrado"))
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	if usuarioID != comentario.AutorID && usuarioID != donoDaPublicacao {
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possível excluir um comentário que não é seu"))
		return
	}

	if erro = repositorio.Deletar(comentarioID); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// AtualizarComentario edita um comentario; so o autor pode fazer isso
func AtualizarComentario(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	comentarioID, erro := strconv.ParseUint(mux.Vars(r)["comentarioId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	var novo modelos.Comentario
	if erro = json.NewDecoder(r.Body).Decode(&novo); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Corpo da requisição inválido"))
		return
	}
	if erro = novo.Preparar(); erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDeComentarios(db)
	existente, _, erro := repositorio.BuscarPorID(comentarioID)
	if erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Comentário não encontrado"))
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	if existente.AutorID != usuarioID {
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possível editar um comentário que não é seu"))
		return
	}

	if erro = repositorio.Atualizar(comentarioID, novo.Conteudo); erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// CurtirComentario e DescurtirComentario alternam a curtida do usuario autenticado
func CurtirComentario(w http.ResponseWriter, r *http.Request) {
	alternarCurtidaDeComentario(w, r, true)
}

func DescurtirComentario(w http.ResponseWriter, r *http.Request) {
	alternarCurtidaDeComentario(w, r, false)
}

func alternarCurtidaDeComentario(w http.ResponseWriter, r *http.Request, curtir bool) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	comentarioID, erro := strconv.ParseUint(mux.Vars(r)["comentarioId"], 10, 64)
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

	repositorio := repositorios.NovoRepositorioDeComentarios(db)
	if curtir {
		erro = repositorio.Curtir(comentarioID, usuarioID)
	} else {
		erro = repositorio.Descurtir(comentarioID, usuarioID)
	}
	if erro != nil {
		var erroMySQL *mysql.MySQLError
		if errors.As(erro, &erroMySQL) && erroMySQL.Number == 1452 {
			respostas.Erro(w, http.StatusNotFound, errors.New("Comentário não encontrado"))
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	registrarNotificacao(db, func(n *repositorios.Notificacoes) error {
		if curtir {
			return n.NotificarCurtidaDeComentario(comentarioID, usuarioID)
		}
		return n.RemoverCurtidaDeComentario(comentarioID, usuarioID)
	})

	respostas.JSON(w, http.StatusNoContent, nil)
}
