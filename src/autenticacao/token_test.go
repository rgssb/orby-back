package autenticacao

import (
	"net/http"
	"net/http/httptest"
	"orby/api/src/config"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

func requisicaoComToken(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func assinar(t *testing.T, metodo jwt.SigningMethod, chave interface{}, claims jwt.MapClaims) string {
	t.Helper()
	token, erro := jwt.NewWithClaims(metodo, claims).SignedString(chave)
	if erro != nil {
		t.Fatal(erro)
	}
	return token
}

func TestTokenValidoDevolveOUsuario(t *testing.T) {
	config.SecretKey = []byte("segredo-de-teste")

	token, erro := CriarToken(42)
	if erro != nil {
		t.Fatal(erro)
	}
	r := requisicaoComToken(token)
	if erro = ValidarToken(r); erro != nil {
		t.Fatalf("token válido recusado: %v", erro)
	}
	if id, erro := ExtrairUsuarioID(r); erro != nil || id != 42 {
		t.Fatalf("id = %d, erro = %v", id, erro)
	}
}

func TestTokenInvalidoEhRecusado(t *testing.T) {
	config.SecretKey = []byte("segredo-de-teste")
	futuro := time.Now().Add(time.Hour).Unix()

	casos := map[string]string{
		"sem cabeçalho":  "",
		"lixo":           "nao-eh-jwt",
		"chave errada":   assinar(t, jwt.SigningMethodHS256, []byte("outra"), jwt.MapClaims{"usuarioID": 1, "exp": futuro}),
		"expirado":       assinar(t, jwt.SigningMethodHS256, config.SecretKey, jwt.MapClaims{"usuarioID": 1, "exp": time.Now().Add(-time.Minute).Unix()}),
		"algoritmo none": assinar(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, jwt.MapClaims{"usuarioID": 1, "exp": futuro}),
	}
	for nome, token := range casos {
		t.Run(nome, func(t *testing.T) {
			r := requisicaoComToken(token)
			if erro := ValidarToken(r); erro == nil {
				t.Error("ValidarToken aceitou um token inválido")
			}
			if _, erro := ExtrairUsuarioID(r); erro == nil {
				t.Error("ExtrairUsuarioID aceitou um token inválido")
			}
		})
	}
}
