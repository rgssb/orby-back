package middlewares

import (
	"net/http"
	"net/http/httptest"
	"orby/api/src/autenticacao"
	"orby/api/src/config"
	"testing"
)

func TestAutenticarBarraRequisicaoSemToken(t *testing.T) {
	config.SecretKey = []byte("segredo-de-teste")
	chamado := false
	handler := Autenticar(func(w http.ResponseWriter, r *http.Request) { chamado = true })

	resposta := httptest.NewRecorder()
	handler(resposta, httptest.NewRequest(http.MethodGet, "/", nil))
	if resposta.Code != http.StatusUnauthorized || chamado {
		t.Fatalf("sem token: status=%d, handler chamado=%v", resposta.Code, chamado)
	}

	token, _ := autenticacao.CriarToken(1)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	resposta = httptest.NewRecorder()
	handler(resposta, r)
	if !chamado {
		t.Fatalf("com token válido o handler deveria rodar (status %d)", resposta.Code)
	}
}
