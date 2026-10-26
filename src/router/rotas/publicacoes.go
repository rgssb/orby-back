package rotas

import (
	"API/src/controllers"
	"net/http"
)

var rotasPublicacoes = []Rota{
	{URI: "/publicacoes/salvas", Metodo: http.MethodGet, Funcao: controllers.BuscarPublicacoesSalvas, RequerAutenticacao: true},
	{URI: "/publicacoes/{publicacaoId}/salvar", Metodo: http.MethodPost, Funcao: controllers.SalvarPublicacao, RequerAutenticacao: true},
	{URI: "/publicacoes/{publicacaoId}/salvar", Metodo: http.MethodDelete, Funcao: controllers.RemoverPublicacaoSalva, RequerAutenticacao: true},
	{
		URI:                "/publicacoes",
		Metodo:             http.MethodPost,
		Funcao:             controllers.CriarPublicacoa,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/busca",
		Metodo:             http.MethodGet,
		Funcao:             controllers.PesquisarPublicacoes,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}/imagem",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarImagemDaPublicacao,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}/imagem",
		Metodo:             http.MethodPost,
		Funcao:             controllers.SalvarImagemDaPublicacao,
		RequerAutenticacao: true,
		LimiteCorpo:        controllers.LimiteImagemPublicacao + 1024,
	},

	{
		URI:                "/publicacoes/{publicacaoId}/imagem",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.RemoverImagemDaPublicacao,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarPublicacoes,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarPublicacoa,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}",
		Metodo:             http.MethodPut,
		Funcao:             controllers.AtualizarPublicacoa,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.DeletarPublicacoa,
		RequerAutenticacao: true,
	},

	{
		URI:                "/usuarios/{usuarioId}/publicacoes",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarPublicacoesPorUsuario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/publicacoes/{publicacaoId}/curtir",
		Metodo:             http.MethodPost,
		Funcao:             controllers.CurtirPublicacoa,
		RequerAutenticacao: true,
	},
	{
		URI:                "/publicacoes/{publicacaoId}/descurtir",
		Metodo:             http.MethodPost,
		Funcao:             controllers.DescurtirPublicacoa,
		RequerAutenticacao: true,
	},
}
