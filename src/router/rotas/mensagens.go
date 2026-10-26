package rotas

import (
	"API/src/controllers"
	"net/http"
)

var rotasMensagens = []Rota{
	{URI: "/conversas", Metodo: http.MethodGet, Funcao: controllers.ListarConversas, RequerAutenticacao: true},
	{URI: "/conversas/nao-lidas", Metodo: http.MethodGet, Funcao: controllers.ContarMensagensNaoLidas, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/mensagens", Metodo: http.MethodGet, Funcao: controllers.BuscarMensagens, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/mensagens", Metodo: http.MethodPost, Funcao: controllers.EnviarMensagem, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/aceitar", Metodo: http.MethodPost, Funcao: controllers.AceitarConversa, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}", Metodo: http.MethodDelete, Funcao: controllers.ExcluirConversa, RequerAutenticacao: true},
	{URI: "/mensagens/{mensagemId}", Metodo: http.MethodDelete, Funcao: controllers.ExcluirMensagem, RequerAutenticacao: true},
}
