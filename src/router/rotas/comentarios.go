package rotas

import (
	"net/http"
	"orby/api/src/controllers"
)

var rotasComentarios = []Rota{
	{
		URI:                "/publicacoes/{publicacaoId}/comentarios",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarComentarios,
		RequerAutenticacao: true,
	},
	{
		URI:                "/publicacoes/{publicacaoId}/comentarios",
		Metodo:             http.MethodPost,
		Funcao:             controllers.CriarComentario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/comentarios/{comentarioId}",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.DeletarComentario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/comentarios/{comentarioId}",
		Metodo:             http.MethodPut,
		Funcao:             controllers.AtualizarComentario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/comentarios/{comentarioId}/curtir",
		Metodo:             http.MethodPost,
		Funcao:             controllers.CurtirComentario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/comentarios/{comentarioId}/descurtir",
		Metodo:             http.MethodPost,
		Funcao:             controllers.DescurtirComentario,
		RequerAutenticacao: true,
	},
}

var rotasRecuperacaoDeSenha = []Rota{
	{
		URI:                "/esqueci-senha",
		Metodo:             http.MethodPost,
		Funcao:             controllers.EsqueciSenha,
		RequerAutenticacao: false,
	},
	{
		URI:                "/redefinir-senha",
		Metodo:             http.MethodPost,
		Funcao:             controllers.RedefinirSenha,
		RequerAutenticacao: false,
	},
}
