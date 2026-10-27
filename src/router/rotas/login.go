package rotas

import (
	"net/http"
	"orby/api/src/controllers"
)

var rotalogin = Rota{

	URI:                "/login",
	Metodo:             http.MethodPost,
	Funcao:             controllers.Login,
	RequerAutenticacao: false,
}
