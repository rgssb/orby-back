package rotas

import (
	"net/http"
	"orby/api/src/controllers"
)

var rotasNotificacoes = []Rota{
	{URI: "/admin/usuarios/{usuarioId}/denuncias", Metodo: http.MethodGet, Funcao: controllers.DetalhesDeDenuncias, RequerAutenticacao: true},
	{URI: "/notificacoes/preferencias", Metodo: http.MethodGet, Funcao: controllers.PreferenciasDeNotificacao, RequerAutenticacao: true},
	{URI: "/notificacoes/preferencias", Metodo: http.MethodPut, Funcao: controllers.PreferenciasDeNotificacao, RequerAutenticacao: true},
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
		URI:                "/denuncias/consulta",
		Metodo:             http.MethodPost,
		Funcao:             controllers.ConsultarDenuncias,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/eu",
		Metodo:             http.MethodGet,
		Funcao:             controllers.SouAdmin,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/painel",
		Metodo:             http.MethodGet,
		Funcao:             controllers.PainelDaAdministracao,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/lista/{tipo}",
		Metodo:             http.MethodGet,
		Funcao:             controllers.ListarAdministracao,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/usuarios/{usuarioId}/suspender",
		Metodo:             http.MethodPost,
		Funcao:             controllers.SuspenderUsuario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/usuarios/{usuarioId}/reativar",
		Metodo:             http.MethodPost,
		Funcao:             controllers.ReativarUsuario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/admin/denuncias/{usuarioId}/resolver",
		Metodo:             http.MethodPost,
		Funcao:             controllers.ResolverDenuncias,
		RequerAutenticacao: true,
	},
}
