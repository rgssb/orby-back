package rotas

import (
	"API/src/controllers"
	"net/http"
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
}
