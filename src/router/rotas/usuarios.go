package rotas

import (
	"API/src/controllers"
	"net/http"
)

var rotasUsuarios = []Rota{
	{
		URI:                "/usuarios",
		Metodo:             http.MethodPost,
		Funcao:             controllers.CriarUsuario,
		RequerAutenticacao: false,
	},
	{
		URI:                "/usuarios",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarUsuarios,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/relacionados",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarUsuariosRelacionados,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/nick/{nick}",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarUsuarioPorNick,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarUsuario,
		RequerAutenticacao: false,
	},
	{
		URI:                "/usuarios/{usuarioId}",
		Metodo:             http.MethodPut,
		Funcao:             controllers.AtualizarUsuario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.DeletarUsuario,
		RequerAutenticacao: true,
	},
	{URI: "/usuarios/{usuarioId}/bloquear", Metodo: http.MethodPost, Funcao: controllers.BloquearUsuario, RequerAutenticacao: true},
	{URI: "/usuarios/{usuarioId}/desbloquear", Metodo: http.MethodPost, Funcao: controllers.DesbloquearUsuario, RequerAutenticacao: true},
	{URI: "/usuarios/{usuarioId}/bloqueio", Metodo: http.MethodGet, Funcao: controllers.SituacaoDeBloqueio, RequerAutenticacao: true},
	{
		URI:                "/usuarios/{usuarioId}/seguir",
		Metodo:             http.MethodPost,
		Funcao:             controllers.SeguirUsuario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/parar-de-seguir",
		Metodo:             http.MethodPost,
		Funcao:             controllers.PararDeSeguirUsuario,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/seguidores",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarSeguidores,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/seguindo",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarSeguindo,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/atualizar-senha",
		Metodo:             http.MethodPost,
		Funcao:             controllers.AtualizarSenha,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/foto",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarFoto,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/foto",
		Metodo:             http.MethodPost,
		Funcao:             controllers.SalvarFoto,
		RequerAutenticacao: true,
		LimiteCorpo:        controllers.LimiteFoto + 1024,
	},
	{
		URI:                "/usuarios/{usuarioId}/foto",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.RemoverFoto,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/banner",
		Metodo:             http.MethodGet,
		Funcao:             controllers.BuscarBanner,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/banner",
		Metodo:             http.MethodPost,
		Funcao:             controllers.SalvarBanner,
		RequerAutenticacao: true,
		LimiteCorpo:        controllers.LimiteFoto + 1024,
	},
	{
		URI:                "/usuarios/{usuarioId}/banner/tema",
		Metodo:             http.MethodPost,
		Funcao:             controllers.DefinirBannerPreset,
		RequerAutenticacao: true,
	},
	{
		URI:                "/usuarios/{usuarioId}/banner",
		Metodo:             http.MethodDelete,
		Funcao:             controllers.RemoverBanner,
		RequerAutenticacao: true,
	},
}
