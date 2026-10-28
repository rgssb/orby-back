package controllers_test

import (
	"net/http"
	"orby/api/src/testes"
	"testing"
)

func TestAdminNaoAlteraPublicacaoDeContaAtiva(t *testing.T) {
	db := testes.NovoBanco(t)
	admin := testes.CriarUsuario(t, db, "admin", false)
	autor := testes.CriarUsuario(t, db, "autor", false)
	if _, erro := db.Exec("update usuarios set admin = 1 where id = ?", admin); erro != nil {
		t.Fatal(erro)
	}
	id := criarPublicacao(t, db, autor)
	if resposta := chamar(t, admin, http.MethodDelete, "/publicacoes/"+id, ""); resposta.Code != http.StatusForbidden {
		t.Fatalf("admin excluiu publicação de conta ativa: %d %s", resposta.Code, resposta.Body.String())
	}
	if resposta := chamar(t, admin, http.MethodPut, "/publicacoes/"+id, `{"conteudo":"alterado"}`); resposta.Code != http.StatusForbidden {
		t.Fatalf("admin alterou publicação de outra conta: %d %s", resposta.Code, resposta.Body.String())
	}
	var quantidade int
	if erro := db.QueryRow("select count(*) from publicacoes where id = ?", id).Scan(&quantidade); erro != nil || quantidade != 1 {
		t.Fatalf("publicação deveria permanecer: %d %v", quantidade, erro)
	}
	if _, erro := db.Exec("update usuarios set suspenso_em = now() where id = ?", autor); erro != nil {
		t.Fatal(erro)
	}
	if resposta := chamar(t, admin, http.MethodDelete, "/publicacoes/"+id, ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("admin não conseguiu remover publicação de conta suspensa: %d %s", resposta.Code, resposta.Body.String())
	}
	propria := criarPublicacao(t, db, admin)
	if resposta := chamar(t, admin, http.MethodDelete, "/publicacoes/"+propria, ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("admin não conseguiu remover a própria publicação: %d %s", resposta.Code, resposta.Body.String())
	}
}
