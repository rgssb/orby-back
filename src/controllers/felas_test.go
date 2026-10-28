package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestFelasProtegePublicacoesEmTodasAsLeituras(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	fela := testes.CriarUsuario(t, db, "fela", false)
	estranho := testes.CriarUsuario(t, db, "estranho", false)
	admin := testes.CriarUsuario(t, db, "admin", false)
	if _, erro := db.Exec("update usuarios set admin = 1 where id = ?", admin); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?), (?, ?)", dono, fela, fela, dono); erro != nil {
		t.Fatal(erro)
	}
	membro := fmt.Sprintf("/felas/%d", fela)
	if r := chamar(t, dono, http.MethodPut, membro, ""); r.Code != http.StatusNoContent {
		t.Fatalf("adicionar fela: %d %s", r.Code, r.Body.String())
	}
	if r := chamar(t, estranho, http.MethodGet, "/felas", ""); strings.Contains(r.Body.String(), "fela") {
		t.Fatalf("outra conta viu a lista: %s", r.Body.String())
	}
	r := chamar(t, dono, http.MethodPost, "/publicacoes", `{"conteudo":"segredo felas @estranho","felas":true}`)
	if r.Code != http.StatusCreated {
		t.Fatalf("criar publicação: %d %s", r.Code, r.Body.String())
	}
	var criada struct{ ID uint64 }
	if erro := json.Unmarshal(r.Body.Bytes(), &criada); erro != nil || criada.ID == 0 {
		t.Fatalf("publicação criada: %s %v", r.Body.String(), erro)
	}
	id := fmt.Sprint(criada.ID)
	for _, caminho := range []string{
		"/publicacoes?aba=todos", "/publicacoes?aba=seguindo", "/publicacoes?aba=felas",
		"/publicacoes/busca?q=segredo", fmt.Sprintf("/usuarios/%d/publicacoes", dono),
		"/publicacoes/salvas", "/notificacoes",
	} {
		r = chamar(t, estranho, http.MethodGet, caminho, "")
		if r.Code != http.StatusOK || strings.Contains(r.Body.String(), "segredo felas") {
			t.Errorf("%s expôs publicação: %d %s", caminho, r.Code, r.Body.String())
		}
	}
	for _, caminho := range []string{"/publicacoes/" + id, "/publicacoes/" + id + "/comentarios", "/publicacoes/" + id + "/imagem"} {
		if r = chamar(t, estranho, http.MethodGet, caminho, ""); r.Code != http.StatusNotFound {
			t.Errorf("acesso indevido a %s: %d", caminho, r.Code)
		}
	}
	for _, acao := range []string{"curtir", "salvar", "comentarios", "repostar"} {
		if r = chamar(t, estranho, http.MethodPost, "/publicacoes/"+id+"/"+acao, `{"conteudo":"oi"}`); r.Code != http.StatusNotFound {
			t.Errorf("ação indevida %s: %d", acao, r.Code)
		}
	}
	comentario, erro := db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'comentário restrito')", criada.ID, dono)
	if erro != nil {
		t.Fatal(erro)
	}
	comentarioID, erro := comentario.LastInsertId()
	if erro != nil {
		t.Fatal(erro)
	}
	for tipo, alvo := range map[string]string{"publicacao": id, "comentario": fmt.Sprint(comentarioID)} {
		if r = chamar(t, estranho, http.MethodPost, "/denuncias", fmt.Sprintf(`{"tipo":%q,"alvoId":%s,"motivo":"spam"}`, tipo, alvo)); r.Code != http.StatusNotFound {
			t.Errorf("denúncia revelou %s inacessível: %d %s", tipo, r.Code, r.Body.String())
		}
	}
	for _, caminho := range []string{"/publicacoes/" + id, "/publicacoes?aba=todos", "/publicacoes?aba=felas", fmt.Sprintf("/usuarios/%d/publicacoes", dono)} {
		if r = chamar(t, fela, http.MethodGet, caminho, ""); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "segredo felas") {
			t.Errorf("fela não viu %s: %d %s", caminho, r.Code, r.Body.String())
		}
	}
	if r = chamar(t, fela, http.MethodPost, "/publicacoes/"+id+"/repostar", `{}`); r.Code != http.StatusForbidden {
		t.Errorf("repost de Felas: %d", r.Code)
	}
	if r = chamar(t, fela, http.MethodPost, "/publicacoes/"+id+"/salvar", ""); r.Code != http.StatusNoContent {
		t.Fatalf("salvar publicação: %d", r.Code)
	}
	if r = chamar(t, admin, http.MethodGet, "/publicacoes/"+id, ""); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"acessoModeracao":true`) {
		t.Fatalf("acesso de moderação: %d %s", r.Code, r.Body.String())
	}
	var acessos int
	if erro := db.QueryRow("select count(*) from acessos_moderacao where admin_id = ? and publicacao_id = ?", admin, criada.ID).Scan(&acessos); erro != nil || acessos == 0 {
		t.Fatalf("acesso administrativo não auditado: %d %v", acessos, erro)
	}
	if r = chamar(t, admin, http.MethodPost, "/publicacoes/"+id+"/curtir", ""); r.Code != http.StatusNotFound {
		t.Errorf("admin interagiu sem acesso comum: %d", r.Code)
	}
	if r = chamar(t, dono, http.MethodDelete, membro, ""); r.Code != http.StatusNoContent {
		t.Fatalf("remover fela: %d", r.Code)
	}
	for _, caminho := range []string{"/publicacoes/" + id, "/publicacoes/salvas", "/notificacoes"} {
		r = chamar(t, fela, http.MethodGet, caminho, "")
		if strings.Contains(r.Body.String(), "segredo felas") {
			t.Errorf("fela removido viu %s: %s", caminho, r.Body.String())
		}
	}
}

func TestFelasRespeitaPerfilPrivadoEBloqueio(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	fela := testes.CriarUsuario(t, db, "fela", false)
	caminho := fmt.Sprintf("/felas/%d", fela)
	if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?), (?, ?)", dono, fela, fela, dono); erro != nil {
		t.Fatal(erro)
	}
	if r := chamar(t, dono, http.MethodPut, caminho, ""); r.Code != http.StatusNoContent {
		t.Fatalf("adicionar: %d %s", r.Code, r.Body.String())
	}
	if _, erro := db.Exec("delete from seguidores where usuario_id = ? and seguidor_id = ?", dono, fela); erro != nil {
		t.Fatal(erro)
	}
	r := chamar(t, dono, http.MethodPost, "/publicacoes", `{"conteudo":"privado","felas":true}`)
	var criada struct{ ID uint64 }
	if r.Code != http.StatusCreated || json.Unmarshal(r.Body.Bytes(), &criada) != nil {
		t.Fatalf("criar: %d %s", r.Code, r.Body.String())
	}

	post := fmt.Sprintf("/publicacoes/%d", criada.ID)
	if r = chamar(t, fela, http.MethodGet, post, ""); r.Code != http.StatusNotFound {
		t.Fatalf("não seguidor acessou perfil privado: %d", r.Code)
	}
	if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", dono, fela); erro != nil {
		t.Fatal(erro)
	}
	if r = chamar(t, fela, http.MethodGet, post, ""); r.Code != http.StatusOK {
		t.Fatalf("seguidor fela sem acesso: %d %s", r.Code, r.Body.String())
	}
	if r = chamar(t, dono, http.MethodPost, fmt.Sprintf("/usuarios/%d/bloquear", fela), ""); r.Code != http.StatusNoContent {
		t.Fatalf("bloquear: %d", r.Code)
	}
	if r = chamar(t, dono, http.MethodPost, fmt.Sprintf("/usuarios/%d/desbloquear", fela), ""); r.Code != http.StatusNoContent {
		t.Fatalf("desbloquear: %d", r.Code)
	}
	if r = chamar(t, dono, http.MethodGet, caminho, ""); !strings.Contains(r.Body.String(), `"fela":false`) {
		t.Fatalf("bloqueio deveria remover a pessoa dos Felas: %s", r.Body.String())
	}
}
