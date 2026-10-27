package rotas

import (
	"orby/api/src/middlewares"
	"github.com/gorilla/mux"
	"net/http"
)

// Rota representa todas as rotas da API
type Rota struct {
	URI                string
	Metodo             string
	Funcao             func(http.ResponseWriter, *http.Request)
	RequerAutenticacao bool
	// LimiteCorpo é o tamanho máximo do corpo da requisição em bytes (0 usa o padrão de 1 MB)
	LimiteCorpo int64
}

const limitePadraoDoCorpo = 1 << 20

// Configurar coloca todas as rotas dentro do router
func Configurar(r *mux.Router) *mux.Router {
	rotas := rotasUsuarios
	rotas = append(rotas, rotalogin)
	rotas = append(rotas, rotasPublicacoes...)
	rotas = append(rotas, rotasComentarios...)
	rotas = append(rotas, rotasRecuperacaoDeSenha...)
	rotas = append(rotas, rotasNotificacoes...)
	rotas = append(rotas, rotasMensagens...)

	for _, rota := range rotas {
		limite := rota.LimiteCorpo
		if limite == 0 {
			limite = limitePadraoDoCorpo
		}

		funcao := middlewares.LimitarCorpo(limite, rota.Funcao)
		if rota.RequerAutenticacao {
			funcao = middlewares.Autenticar(funcao)
		}

		r.HandleFunc(rota.URI, middlewares.Logger(funcao)).Methods(rota.Metodo)
	}

	return r

}
