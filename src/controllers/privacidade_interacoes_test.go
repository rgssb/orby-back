package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestPrivacidadeDeInteracoes(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	seguido := testes.CriarUsuario(t, db, "seguido", false)
	estranho := testes.CriarUsuario(t, db, "estranho", false)
	seguidor := testes.CriarUsuario(t, db, "seguidor", false)
	if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?), (?, ?)", seguido, dono, dono, seguidor); erro != nil {
		t.Fatal(erro)
	}
	direta := fmt.Sprintf("/conversas/%d/mensagens", dono)
	if r := chamar(t, estranho, http.MethodPost, direta, `{"texto":"antes"}`); r.Code != http.StatusCreated {
		t.Fatalf("conversa inicial: %d %s", r.Code, r.Body.String())
	}
	r := chamar(t, dono, http.MethodPost, "/grupos", fmt.Sprintf(`{"nome":"Amigos","membros":[%d,%d]}`, seguido, estranho))
	var grupo struct{ ID uint64 }
	if r.Code != http.StatusCreated || json.Unmarshal(r.Body.Bytes(), &grupo) != nil || grupo.ID == 0 {
		t.Fatalf("grupo inicial: %d %s", r.Code, r.Body.String())
	}
	rota := "/privacidade/interacoes"
	if r = chamar(t, dono, http.MethodGet, rota, ""); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"mensagens":"todos"`) {
		t.Fatalf("padrão: %d %s", r.Code, r.Body.String())
	}
	if r = chamar(t, dono, http.MethodPut, rota, `{"mensagens":"seguindo","comentarios":"seguindo","mencoes":"seguindo"}`); r.Code != http.StatusNoContent {
		t.Fatalf("salvar: %d %s", r.Code, r.Body.String())
	}
	if r = chamar(t, dono, http.MethodPut, rota, `{"mensagens":"todos","comentarios":"invalido","mencoes":"todos"}`); r.Code != http.StatusBadRequest {
		t.Fatalf("valor inválido: %d", r.Code)
	}
	if r = chamar(t, estranho, http.MethodPost, direta, `{"texto":"depois"}`); r.Code != http.StatusForbidden {
		t.Errorf("conversa existente não respeitou privacidade: %d %s", r.Code, r.Body.String())
	}
	if r = chamar(t, seguido, http.MethodPost, direta, `{"texto":"permitido"}`); r.Code != http.StatusCreated {
		t.Errorf("conta seguida sem acesso: %d %s", r.Code, r.Body.String())
	}
	grupoMensagens := fmt.Sprintf("/grupos/%d/mensagens", grupo.ID)
	if r = chamar(t, estranho, http.MethodPost, grupoMensagens, `{"texto":"negado"}`); r.Code != http.StatusForbidden {
		t.Errorf("grupo não respeitou privacidade: %d %s", r.Code, r.Body.String())
	}
	if r = chamar(t, seguido, http.MethodPost, grupoMensagens, `{"texto":"permitido"}`); r.Code != http.StatusCreated {
		t.Errorf("grupo recusou conta seguida: %d %s", r.Code, r.Body.String())
	}
	post := criarPublicacao(t, db, dono)
	comentarios := "/publicacoes/" + post + "/comentarios"
	if r = chamar(t, estranho, http.MethodPost, comentarios, `{"conteudo":"negado"}`); r.Code != http.StatusForbidden {
		t.Errorf("comentário não respeitou privacidade: %d %s", r.Code, r.Body.String())
	}
	if r = chamar(t, seguido, http.MethodPost, comentarios, `{"conteudo":"permitido"}`); r.Code != http.StatusCreated {
		t.Errorf("comentário de seguido recusado: %d %s", r.Code, r.Body.String())
	}
	for _, pessoa := range []uint64{seguido, estranho} {
		r = chamar(t, pessoa, http.MethodPost, "/publicacoes", `{"conteudo":"oi @dono"}`)
		if r.Code != http.StatusCreated {
			t.Fatalf("publicar menção: %d %s", r.Code, r.Body.String())
		}
	}
	var avisosSeguido, avisosEstranho int
	if erro := db.QueryRow("select count(*) from notificacoes where usuario_id = ? and ator_id = ? and tipo = 'mencao'", dono, seguido).Scan(&avisosSeguido); erro != nil {
		t.Fatal(erro)
	}
	if erro := db.QueryRow("select count(*) from notificacoes where usuario_id = ? and ator_id = ? and tipo = 'mencao'", dono, estranho).Scan(&avisosEstranho); erro != nil {
		t.Fatal(erro)
	}
	if avisosSeguido != 1 || avisosEstranho != 0 {
		t.Errorf("avisos de menção: seguido=%d estranho=%d", avisosSeguido, avisosEstranho)
	}
	for _, caso := range []struct {
		viewer uint64
		link   bool
	}{{dono, true}, {seguido, true}, {seguidor, true}, {estranho, false}} {
		r = chamar(t, caso.viewer, http.MethodPost, "/mencoes/visibilidade", `{"nicks":["dono"]}`)
		var visibilidade map[string]bool
		if r.Code != http.StatusOK || json.Unmarshal(r.Body.Bytes(), &visibilidade) != nil || visibilidade["dono"] != caso.link {
			t.Errorf("visibilidade para %d: %d %s", caso.viewer, r.Code, r.Body.String())
		}
	}
	if r = chamar(t, dono, http.MethodPut, rota, `{"mensagens":"ninguem","comentarios":"ninguem","mencoes":"ninguem"}`); r.Code != http.StatusNoContent {
		t.Fatalf("restringir: %d %s", r.Code, r.Body.String())
	}
	if r = chamar(t, seguido, http.MethodPost, direta, `{"texto":"negado"}`); r.Code != http.StatusForbidden {
		t.Errorf("ninguém deveria enviar mensagem: %d", r.Code)
	}
	if r = chamar(t, seguido, http.MethodPost, comentarios, `{"conteudo":"negado"}`); r.Code != http.StatusForbidden {
		t.Errorf("ninguém deveria comentar: %d", r.Code)
	}
	if r = chamar(t, seguido, http.MethodPost, "/mencoes/visibilidade", `{"nicks":["dono"]}`); !strings.Contains(r.Body.String(), `"dono":false`) {
		t.Errorf("menção restrita apareceu como link: %s", r.Body.String())
	}
}
