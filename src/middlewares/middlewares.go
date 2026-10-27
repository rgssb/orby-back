package middlewares

import (
	"orby/api/src/autenticacao"
	"orby/api/src/respostas"
	"log"
	"net/http"
)

// Logger escreve as informacoes da requisicao no terminal
func Logger(proximaFuncao http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("\n %s %s %s", r.Method, r.RequestURI, r.Host)
		proximaFuncao(w, r)
	}
}

// LimitarCorpo recusa corpos de requisição maiores que o limite informado
func LimitarCorpo(limite int64, proximaFuncao http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, limite)
		proximaFuncao(w, r)
	}
}

// func (w http.ResponseWriter, r *http.Request)
// Autenticar verifica se o usuario que esta fazendo a autenticação, esta autenticado
func Autenticar(proximaFuncao http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if erro := autenticacao.ValidarToken(r); erro != nil {
			respostas.Erro(w, http.StatusUnauthorized, erro)
			return
		}
		proximaFuncao(w, r)
	}
}
