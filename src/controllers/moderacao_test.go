package controllers_test

import (
	"fmt"
	"net/http"
	"orby/api/src/repositorios"
	"orby/api/src/testes"
	"strconv"
	"strings"
	"testing"
)

func TestModeracaoDePerfilPrivado(t *testing.T) {
	db := testes.NovoBanco(t)
	admin := testes.CriarUsuario(t, db, "admin", false)
	dono := testes.CriarUsuario(t, db, "privado", true)
	visitante := testes.CriarUsuario(t, db, "visitante", false)
	if _, erro := db.Exec("update usuarios set admin = 1 where id = ?", admin); erro != nil {
		t.Fatal(erro)
	}
	id := criarPublicacao(t, db, dono)
	publicacaoID, erro := strconv.ParseUint(id, 10, 64)
	if erro != nil {
		t.Fatal(erro)
	}
	if erro = repositorios.NovoRepositorioDePublicacoes(db).AdicionarImagem(publicacaoID, "image/png", []byte("imagem privada")); erro != nil {
		t.Fatal(erro)
	}
	resultado, erro := db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'comentário privado')", id, dono)
	if erro != nil {
		t.Fatal(erro)
	}
	comentarioID, _ := resultado.LastInsertId()

	perfil := fmt.Sprintf("/usuarios/%d", dono)
	if r := chamar(t, admin, http.MethodGet, perfil, ""); r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"acessoModeracao":true`) || strings.Contains(r.Body.String(), `"email"`) {
		t.Fatalf("perfil administrativo: %d %s", r.Code, r.Body.String())
	}
	if r := chamar(t, visitante, http.MethodGet, perfil, ""); strings.Contains(r.Body.String(), `"podeVerPublicacoes":true`) {
		t.Fatalf("visitante obteve acesso: %s", r.Body.String())
	}
	leituras := []string{
		perfil + "/publicacoes", "/publicacoes/" + id, "/publicacoes/" + id + "/imagem",
		"/publicacoes/" + id + "/comentarios",
		fmt.Sprintf("/comentarios/%d/respostas", comentarioID),
		"/admin/lista/publicacoes?deslocamento=0",
		"/admin/lista/comentarios?deslocamento=0",
	}
	for _, caminho := range leituras {
		if r := chamar(t, admin, http.MethodGet, caminho, ""); r.Code != http.StatusOK {
			t.Errorf("leitura %s: %d %s", caminho, r.Code, r.Body.String())
		}
	}
	var acessos int
	if erro = db.QueryRow("select count(*) from acessos_moderacao where admin_id = ? and alvo_id = ?", admin, dono).Scan(&acessos); erro != nil || acessos != len(leituras) {
		t.Fatalf("acessos registrados: %d, erro: %v", acessos, erro)
	}
	acoes := []struct{ caminho, corpo string }{
		{"/publicacoes/" + id + "/curtir", ""},
		{"/publicacoes/" + id + "/comentarios", `{"conteudo":"teste"}`},
		{fmt.Sprintf("/comentarios/%d/curtir", comentarioID), ""},
	}
	for _, acao := range acoes {
		if r := chamar(t, admin, http.MethodPost, acao.caminho, acao.corpo); r.Code < 400 {
			t.Errorf("moderação permitiu interação %s: %d", acao.caminho, r.Code)
		}
	}
	if _, erro = repositorios.NovoRepositorioDeMensagens(db).Enviar(dono, visitante, "mensagem particular", 0, false); erro != nil {
		t.Fatal(erro)
	}
	if r := chamar(t, admin, http.MethodGet, fmt.Sprintf("/conversas/%d/mensagens", dono), ""); strings.Contains(r.Body.String(), "mensagem particular") {
		t.Fatal("administrador acessou mensagem de terceiros")
	}

	if _, erro = db.Exec("drop table acessos_moderacao"); erro != nil {
		t.Fatal(erro)
	}
	for _, caminho := range []string{perfil + "/publicacoes", "/publicacoes/" + id, "/admin/lista/publicacoes?deslocamento=0"} {
		if r := chamar(t, admin, http.MethodGet, caminho, ""); r.Code != http.StatusInternalServerError || strings.Contains(r.Body.String(), "conteudo secreto") {
			t.Errorf("falha da auditoria não bloqueou %s: %d %s", caminho, r.Code, r.Body.String())
		}
	}
}
