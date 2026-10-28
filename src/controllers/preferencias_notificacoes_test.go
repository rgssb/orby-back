package controllers_test

import (
	"encoding/json"
	"net/http"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestPreferenciasDeNotificacao(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	outro := testes.CriarUsuario(t, db, "outro", false)
	for _, tipo := range []string{"curtida_publicacao", "perfil_alerta"} {
		if _, erro := db.Exec("insert into notificacoes (usuario_id, ator_id, tipo) values (?, ?, ?)", dono, outro, tipo); erro != nil {
			t.Fatal(erro)
		}
	}
	caminho := "/notificacoes/preferencias"
	if r := chamar(t, 0, http.MethodGet, caminho, ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("sem token: %d", r.Code)
	}
	if r := chamar(t, dono, http.MethodPut, caminho, `{"silenciados":["curtida_publicacao"]}`); r.Code != http.StatusNoContent {
		t.Fatalf("salvar: %d %s", r.Code, r.Body.String())
	}
	r := chamar(t, outro, http.MethodGet, caminho, "")
	if !strings.Contains(r.Body.String(), `"silenciados":[]`) {
		t.Fatalf("preferências vazaram para outra conta: %s", r.Body.String())
	}
	for _, corpo := range []string{`{"silenciados":["perfil_alerta"]}`, `{"silenciados":["curtida_publicacao","curtida_publicacao"]}`, `{}`} {
		if r := chamar(t, dono, http.MethodPut, caminho, corpo); r.Code != http.StatusBadRequest {
			t.Fatalf("preferências inválidas %s: %d", corpo, r.Code)
		}
	}
	r = chamar(t, dono, http.MethodGet, "/notificacoes", "")
	var lista []modelos.Notificacao
	if erro := json.Unmarshal(r.Body.Bytes(), &lista); erro != nil || len(lista) != 1 || lista[0].Tipo != "perfil_alerta" {
		t.Fatalf("lista: %s erro=%v", r.Body.String(), erro)
	}
	r = chamar(t, dono, http.MethodGet, "/notificacoes/nao-lidas", "")
	if !strings.Contains(r.Body.String(), `"total":1`) {
		t.Fatalf("contador: %s", r.Body.String())
	}
	if r = chamar(t, dono, http.MethodPost, "/notificacoes/lidas", ""); r.Code != http.StatusNoContent {
		t.Fatalf("marcar como lidas: %d", r.Code)
	}
	if r = chamar(t, dono, http.MethodPut, caminho, `{"silenciados":[]}`); r.Code != http.StatusNoContent {
		t.Fatalf("reativar: %d", r.Code)
	}
	r = chamar(t, dono, http.MethodGet, "/notificacoes/nao-lidas", "")
	if !strings.Contains(r.Body.String(), `"total":1`) {
		t.Fatalf("notificação reativada deveria continuar não lida: %s", r.Body.String())
	}
}
