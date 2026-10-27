package rotas

import (
	"net/http"
	"orby/api/src/controllers"
)

var rotasNotificacoes = []Rota{
	{
		URI:                "/notificacoes",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarNotificacoes,
		RequerAutenticacao: true,
	},
	{
		URI:                "/notificacoes/nao-lidas",
		Metodo:             http.MethodGet,
		Funcao:             controllers.ContarNotificacoesNaoLidas,
		RequerAutenticacao: true,
	},
	{
		URI:                "/notificacoes/{notificacaoId}",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.ExcluirNotificacao,
		RequerAutenticacao: true,
	},
	{
		URI:                "/notificacoes/lidas",
		Metodo:             http.MethodPost,
		Funcao:             controllers.MarcarNotificacoesComoLidas,
		RequerAutenticacao: true,
	},
	{
		URI:                "/denuncias",
		Metodo:             http.MethodPost,
		Funcao:             controllers.Denunciar,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/denuncias",
		Metodo:             http.MethodGet,
		Funcao:             controllers.ListarPerfisDenunciados,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/denuncias/{usuarioId}/resolver",
		Metodo:             http.MethodPost,
		Funcao:             controllers.ResolverDenuncias,
		RequerAutenticacao: true,
	},
}
