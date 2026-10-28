package controllers_test

import (
	"fmt"
	"net/http"
	"orby/api/src/suspensoes"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestDetalhesEHistoricoDeModeracao(t *testing.T) {
	db := testes.NovoBanco(t)
	admin := testes.CriarUsuario(t, db, "admin", false)
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	denunciante := testes.CriarUsuario(t, db, "denunciante", false)
	t.Cleanup(func() { suspensoes.Definir(alvo, false) })
	if _, erro := db.Exec("update usuarios set admin = 1 where id = ?", admin); erro != nil {
		t.Fatal(erro)
	}
	post := criarPublicacao(t, db, alvo)
	corpo := fmt.Sprintf(`{"tipo":"publicacao","alvoId":%s,"motivo":"spam"}`, post)
	if r := chamar(t, denunciante, http.MethodPost, "/denuncias", corpo); r.Code != http.StatusOK {
		t.Fatalf("denunciar: %d %s", r.Code, r.Body.String())
	}
	caminho := fmt.Sprintf("/admin/usuarios/%d/denuncias?deslocamento=0", alvo)
	if r := chamar(t, denunciante, http.MethodGet, caminho, ""); r.Code != http.StatusForbidden {
		t.Fatalf("usuário comum acessou detalhes: %d", r.Code)
	}
	r := chamar(t, admin, http.MethodGet, caminho, "")
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"motivo":"spam"`) ||
		strings.Contains(r.Body.String(), "denunciante") || strings.Contains(r.Body.String(), "conteudo secreto") {
		t.Fatalf("detalhes inválidos ou com dados desnecessários: %d %s", r.Code, r.Body.String())
	}
	for _, acao := range []string{"suspender", "reativar"} {
		rota := fmt.Sprintf("/admin/usuarios/%d/%s", alvo, acao)
		if r = chamar(t, admin, http.MethodPost, rota, ""); r.Code != http.StatusNoContent {
			t.Fatalf("%s: %d %s", acao, r.Code, r.Body.String())
		}
	}
	r = chamar(t, admin, http.MethodGet, caminho, "")
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"resolvida":true`) ||
		!strings.Contains(r.Body.String(), `"acao":"suspender"`) || !strings.Contains(r.Body.String(), `"acao":"reativar"`) {
		t.Fatalf("histórico ausente: %d %s", r.Code, r.Body.String())
	}
}

func TestFalhaDoHistoricoNaoSuspendeConta(t *testing.T) {
	db := testes.NovoBanco(t)
	admin := testes.CriarUsuario(t, db, "admin", false)
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	t.Cleanup(func() { suspensoes.Definir(alvo, false) })
	if _, erro := db.Exec("update usuarios set admin = 1 where id = ?", admin); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := db.Exec("drop table acoes_moderacao"); erro != nil {
		t.Fatal(erro)
	}
	r := chamar(t, admin, http.MethodPost, fmt.Sprintf("/admin/usuarios/%d/suspender", alvo), "")
	if r.Code != http.StatusInternalServerError {
		t.Fatalf("deveria falhar quando não é possível registrar: %d %s", r.Code, r.Body.String())
	}
	var suspensa bool
	if erro := db.QueryRow("select suspenso_em is not null from usuarios where id = ?", alvo).Scan(&suspensa); erro != nil || suspensa {
		t.Fatalf("suspensão não foi revertida: %v %v", suspensa, erro)
	}
}
