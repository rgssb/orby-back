package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"orby/api/src/autenticacao"
	"orby/api/src/modelos"
	"orby/api/src/router"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestDescricaoImagemUploadEdicaoEPermissoes(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	outro := testes.CriarUsuario(t, db, "outro", false)
	id := criarPublicacao(t, db, autor)
	token, erro := autenticacao.CriarToken(autor)
	if erro != nil {
		t.Fatal(erro)
	}
	enviar := func(descricao string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, "/publicacoes/"+id+"/imagens", strings.NewReader("\x89PNG\r\n\x1a\nimagem"))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Orby-Descricao", url.PathEscape(descricao))
		resposta := httptest.NewRecorder()
		router.Gerar().ServeHTTP(resposta, r)
		return resposta
	}
	if resposta := enviar(strings.Repeat("x", 501)); resposta.Code != http.StatusBadRequest {
		t.Fatalf("descrição longa deveria ser recusada: %d", resposta.Code)
	}
	if resposta := enviar("Um cão no jardim"); resposta.Code != http.StatusNoContent {
		t.Fatalf("upload: %d %s", resposta.Code, resposta.Body.String())
	}
	caminho := "/publicacoes/" + id + "/imagens/0/descricao"
	if resposta := chamar(t, outro, http.MethodPut, caminho, `{"descricao":"alterado"}`); resposta.Code != http.StatusForbidden {
		t.Fatalf("outra pessoa não pode alterar: %d", resposta.Code)
	}
	if resposta := chamar(t, autor, http.MethodPut, caminho, `{"descricao":"Um gato na janela"}`); resposta.Code != http.StatusNoContent {
		t.Fatalf("atualizar descrição: %d %s", resposta.Code, resposta.Body.String())
	}
	resposta := chamar(t, autor, http.MethodGet, "/publicacoes/"+id, "")
	var publicacao modelos.Publicacao
	if erro = json.Unmarshal(resposta.Body.Bytes(), &publicacao); erro != nil || len(publicacao.DescricoesImagens) != 1 || publicacao.DescricoesImagens[0] != "Um gato na janela" {
		t.Fatalf("descrição não retornada: %s (%v)", resposta.Body.String(), erro)
	}
}
