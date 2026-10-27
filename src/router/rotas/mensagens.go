package rotas

import (
	"net/http"
	"orby/api/src/controllers"
)

var rotasMensagens = []Rota{
	{URI: "/conversas", Metodo: http.MethodGet, Funcao: controllers.ListarConversas, RequerAutenticacao: true},
	{URI: "/conversas/nao-lidas", Metodo: http.MethodGet, Funcao: controllers.ContarMensagensNaoLidas, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/mensagens", Metodo: http.MethodGet, Funcao: controllers.BuscarMensagens, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/mensagens", Metodo: http.MethodPost, Funcao: controllers.EnviarMensagem, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/aceitar", Metodo: http.MethodPost, Funcao: controllers.AceitarConversa, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}", Metodo: http.MethodDelete, Funcao: controllers.ExcluirConversa, RequerAutenticacao: true},
	{URI: "/grupos", Metodo: http.MethodGet, Funcao: controllers.ListarGrupos, RequerAutenticacao: true},
	{URI: "/grupos", Metodo: http.MethodPost, Funcao: controllers.CriarGrupo, RequerAutenticacao: true},
	{URI: "/grupos/{grupoId}", Metodo: http.MethodGet, Funcao: controllers.BuscarGrupo, RequerAutenticacao: true},
	{URI: "/grupos/{grupoId}/mensagens", Metodo: http.MethodPost, Funcao: controllers.EnviarMensagemDeGrupo, RequerAutenticacao: true},
	{URI: "/grupos/{grupoId}/membros", Metodo: http.MethodPost, Funcao: controllers.AdicionarAoGrupo, RequerAutenticacao: true},
	{URI: "/grupos/{grupoId}/membros/{usuarioId}/permissao", Metodo: http.MethodPut, Funcao: controllers.DefinirPermissaoNoGrupo, RequerAutenticacao: true},
	{URI: "/grupos/{grupoId}/membros/{usuarioId}", Metodo: http.MethodDelete, Funcao: controllers.RemoverDoGrupo, RequerAutenticacao: true},
	{URI: "/mensagens/{mensagemId}", Metodo: http.MethodDelete, Funcao: controllers.ExcluirMensagem, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/silenciar", Metodo: http.MethodPost, Funcao: controllers.SilenciarConversa, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/silenciar", Metodo: http.MethodDelete, Funcao: controllers.DessilenciarConversa, RequerAutenticacao: true},
	{URI: "/conversas/{usuarioId}/digitando", Metodo: http.MethodPost, Funcao: controllers.AvisarDigitacao, RequerAutenticacao: true},
	{URI: "/mensagens/{mensagemId}/reacao", Metodo: http.MethodPost, Funcao: controllers.ReagirAMensagem, RequerAutenticacao: true},
	{URI: "/mensagens/{mensagemId}/imagem", Metodo: http.MethodPost, Funcao: controllers.EnviarImagemDeMensagem, RequerAutenticacao: true, LimiteCorpo: controllers.LimiteImagemMensagem + 1024},
	{URI: "/mensagens/{mensagemId}/imagem", Metodo: http.MethodGet, Funcao: controllers.BuscarImagemDeMensagem, RequerAutenticacao: true},
	{URI: "/eventos", Metodo: http.MethodGet, Funcao: controllers.Eventos, RequerAutenticacao: true},
}
