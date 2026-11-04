package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestTituloOpcionalDaEnquete(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	for _, titulo := range []string{"", "  Qual passeio?  "} {
		corpo, erro := json.Marshal(map[string]interface{}{
			"conteudo": "Olá",
			"enquete":  map[string]interface{}{"titulo": titulo, "opcoes": []map[string]string{{"texto": "Praia"}, {"texto": "Serra"}}},
		})
		if erro != nil {
			t.Fatal(erro)
		}
		resposta := chamar(t, autor, http.MethodPost, "/publicacoes", string(corpo))
		var criada modelos.Publicacao
		if resposta.Code != http.StatusCreated || json.Unmarshal(resposta.Body.Bytes(), &criada) != nil {
			t.Fatalf("criação: %d %s", resposta.Code, resposta.Body)
		}
		resposta = chamar(t, autor, http.MethodGet, fmt.Sprintf("/publicacoes/%d", criada.ID), "")
		var buscada modelos.Publicacao
		if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &buscada) != nil ||
			buscada.Enquete == nil || buscada.Enquete.Titulo != strings.TrimSpace(titulo) {
			t.Fatalf("título retornado incorretamente: %d %s", resposta.Code, resposta.Body)
		}
	}
	corpo, erro := json.Marshal(map[string]interface{}{
		"conteudo": "Olá",
		"enquete":  map[string]interface{}{"titulo": strings.Repeat("x", 151), "opcoes": []map[string]string{{"texto": "a"}, {"texto": "b"}}},
	})
	if erro != nil {
		t.Fatal(erro)
	}
	resposta := chamar(t, autor, http.MethodPost, "/publicacoes", string(corpo))
	if resposta.Code != http.StatusBadRequest {
		t.Fatalf("título longo aceito: %d %s", resposta.Code, resposta.Body)
	}
}
