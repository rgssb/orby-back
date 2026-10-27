package rotas

import (
	"net/http"
	"orby/api/src/controllers"
)

var rotasPublicacoes = []Rota{
	{URI: "/publicacoes/salvas", Metodo: http.MethodGet, Funcao: controllers.BuscarPublicacoesSalvas, RequerAutenticacao: true},
	{URI: "/publicacoes/{publicacaoId}/salvar", Metodo: http.MethodPost, Funcao: controllers.SalvarPublicacao, RequerAutenticacao: true},
	{URI: "/publicacoes/{publicacaoId}/salvar", Metodo: http.MethodDelete, Funcao: controllers.RemoverPublicacaoSalva, RequerAutenticacao: true},
	{
		URI:                "/publicacoes",
		Metodo:             http.MethodPost,
		Funcao:             controllers.CriarPublicacao,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/busca",
		Metodo:             http.MethodGet,
		Funcao:             controllers.PesquisarPublicacoes,
		RequerAutenticacao: true,
	},

	{URI: "/publicacoes/{publicacaoId}/imagem", Metodo: http.MethodGet, Funcao: controllers.BuscarImagemDaPublicacao, RequerAutenticacao: true},
	{URI: "/publicacoes/{publicacaoId}/imagens/{posicao}", Metodo: http.MethodGet, Funcao: controllers.BuscarImagemDaPublicacao, RequerAutenticacao: true},
	{
		URI:                "/publicacoes/{publicacaoId}/imagens",
		Metodo:             http.MethodPost,
		Funcao:             controllers.AdicionarImagemDaPublicacao,
		RequerAutenticacao: true,
		LimiteCorpo:        controllers.LimiteImagemPublicacao + 1024,
	},
	{URI: "/publicacoes/{publicacaoId}/imagens/{posicao}", Metodo: http.MethodDelete, Funcao: controllers.RemoverImagemDaPublicacao, RequerAutenticacao: true},

	{
		URI:                "/publicacoes",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarPublicacoes,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarPublicacao,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}",
		Metodo:             http.MethodPut,
		Funcao:             controllers.AtualizarPublicacao,
		RequerAutenticacao: true,
	},

	{
		URI:                "/publicacoes/{publicacaoId}",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.DeletarPublicacao,
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
		Funcao:             controllers.CurtirPublicacao,
		RequerAutenticacao: true,
	},
	{
		URI:                "/publicacoes/{publicacaoId}/repostar",
		Metodo:             http.MethodPost,
		Funcao:             controllers.Repostar,
		RequerAutenticacao: true,
	},
	{
		URI:                "/publicacoes/{publicacaoId}/repostar",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.DesfazerRepost,
		RequerAutenticacao: true,
	},
	{
		URI:                "/publicacoes/{publicacaoId}/descurtir",
		Metodo:             http.MethodPost,
		Funcao:             controllers.DescurtirPublicacao,
		RequerAutenticacao: true,
	},
}
