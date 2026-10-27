package controllers_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"orby/api/src/autenticacao"
	"orby/api/src/router"
	"orby/api/src/testes"
	"strconv"
	"strings"
	"testing"
)

// chamar executa uma requisição no router completo como o usuário informado (0 = sem token)
func chamar(t *testing.T, usuarioID uint64, metodo, caminho, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	if usuarioID != 0 {
		token, erro := autenticacao.CriarToken(usuarioID)
		if erro != nil {
			t.Fatal(erro)
		}
		r.Header.Set("Authorization", "Bearer "+token)
	}
	resposta := httptest.NewRecorder()
	router.Gerar().ServeHTTP(resposta, r)
	return resposta
}

func criarPublicacao(t *testing.T, db *sql.DB, autorID uint64) string {
	t.Helper()
	resultado, erro := db.Exec("insert into publicacoes (titulo, conteudo, autor_id) values ('titulo', 'conteudo secreto', ?)", autorID)
	if erro != nil {
		t.Fatal(erro)
	}
	id, _ := resultado.LastInsertId()
	return strconv.FormatInt(id, 10)
}

func TestRotasProtegidasExigemToken(t *testing.T) {
	testes.NovoBanco(t)
	for _, rota := range [][2]string{
		{http.MethodGet, "/publicacoes"},
		{http.MethodGet, "/publicacoes/1"},
		{http.MethodDelete, "/publicacoes/1"},
		{http.MethodGet, "/conversas"},
		{http.MethodPost, "/usuarios/1/bloquear"},
		{http.MethodGet, "/eventos"},
	} {
		if resposta := chamar(t, 0, rota[0], rota[1], ""); resposta.Code != http.StatusUnauthorized {
			t.Errorf("%s %s sem token: status %d", rota[0], rota[1], resposta.Code)
		}
	}
}

func TestSoOAutorEditaOuApagaPublicacao(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	intruso := testes.CriarUsuario(t, db, "intruso", false)
	id := criarPublicacao(t, db, autor)
	corpo := `{"titulo":"novo","conteudo":"novo conteudo"}`

	if resposta := chamar(t, intruso, http.MethodPut, "/publicacoes/"+id, corpo); resposta.Code != http.StatusForbidden {
		t.Errorf("editar publicação alheia: status %d", resposta.Code)
	}
	if resposta := chamar(t, intruso, http.MethodDelete, "/publicacoes/"+id, ""); resposta.Code != http.StatusForbidden {
		t.Errorf("apagar publicação alheia: status %d", resposta.Code)
	}
	var titulo string
	db.QueryRow("select titulo from publicacoes where id = ?", id).Scan(&titulo)
	if titulo != "titulo" {
		t.Fatalf("a publicação alheia foi alterada ou apagada (titulo = %q)", titulo)
	}

	if resposta := chamar(t, autor, http.MethodPut, "/publicacoes/"+id, corpo); resposta.Code != http.StatusNoContent {
		t.Errorf("editar a própria publicação: status %d", resposta.Code)
	}
	if resposta := chamar(t, autor, http.MethodDelete, "/publicacoes/"+id, ""); resposta.Code != http.StatusNoContent {
		t.Errorf("apagar a própria publicação: status %d", resposta.Code)
	}
}

func TestSoOAutorOuODonoApagaComentario(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	autor := testes.CriarUsuario(t, db, "autor", false)
	intruso := testes.CriarUsuario(t, db, "intruso", false)
	publicacao := criarPublicacao(t, db, dono)

	resposta := chamar(t, autor, http.MethodPost, "/publicacoes/"+publicacao+"/comentarios", `{"conteudo":"oi"}`)
	if resposta.Code != http.StatusCreated {
		t.Fatalf("criar comentário: status %d", resposta.Code)
	}
	var comentario struct{ ID uint64 }
	json.Unmarshal(resposta.Body.Bytes(), &comentario)
	caminho := "/comentarios/" + strconv.FormatUint(comentario.ID, 10)

	if resposta = chamar(t, intruso, http.MethodDelete, caminho, ""); resposta.Code == http.StatusNoContent {
		t.Fatal("um terceiro apagou o comentário")
	}
	if resposta = chamar(t, intruso, http.MethodPut, caminho, `{"conteudo":"editado"}`); resposta.Code == http.StatusNoContent || resposta.Code == http.StatusOK {
		t.Fatal("um terceiro editou o comentário")
	}
	if resposta = chamar(t, dono, http.MethodDelete, caminho, ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("o dono da publicação deveria apagar: status %d", resposta.Code)
	}
}

func TestPublicacaoDePerfilPrivadoNaoVazaPelasRotas(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	seguidor := testes.CriarUsuario(t, db, "seguidor", false)
	estranho := testes.CriarUsuario(t, db, "estranho", false)
	db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", dono, seguidor)
	id := criarPublicacao(t, db, dono)
	db.Exec("insert into imagens_publicacoes (publicacao_id, tipo) values (?, 'image/png')", id)

	leituras := []struct{ metodo, caminho, corpo string }{
		{http.MethodGet, "/publicacoes/" + id, ""},
		{http.MethodGet, "/publicacoes/" + id + "/comentarios", ""},
		{http.MethodGet, "/publicacoes/" + id + "/imagem", ""},
		{http.MethodGet, "/usuarios/" + strconv.FormatUint(dono, 10) + "/publicacoes", ""},
	}
	for _, l := range leituras {
		if resposta := chamar(t, estranho, l.metodo, l.caminho, l.corpo); strings.Contains(resposta.Body.String(), "conteudo secreto") || resposta.Code == http.StatusOK && !strings.HasSuffix(l.caminho, "/publicacoes") {
			t.Errorf("estranho acessou %s: status %d", l.caminho, resposta.Code)
		}
	}

	acoes := []struct{ metodo, caminho, corpo string }{
		{http.MethodPost, "/publicacoes/" + id + "/comentarios", `{"conteudo":"intruso"}`},
		{http.MethodPost, "/publicacoes/" + id + "/curtir", ""},
		{http.MethodPost, "/publicacoes/" + id + "/salvar", ""},
	}
	for _, a := range acoes {
		if resposta := chamar(t, estranho, a.metodo, a.caminho, a.corpo); resposta.Code < 400 {
			t.Errorf("estranho conseguiu %s %s: status %d", a.metodo, a.caminho, resposta.Code)
		}
	}
	var interacoes int
	db.QueryRow("select (select count(*) from comentarios) + (select count(*) from curtidas_publicacoes) + (select count(*) from salvos)").Scan(&interacoes)
	if interacoes != 0 {
		t.Errorf("o estranho deixou %d interações em um perfil privado", interacoes)
	}

	if resposta := chamar(t, seguidor, http.MethodGet, "/publicacoes/"+id, ""); resposta.Code != http.StatusOK {
		t.Errorf("seguidor deveria ver a publicação: status %d", resposta.Code)
	}
	if resposta := chamar(t, dono, http.MethodGet, "/publicacoes/"+id, ""); resposta.Code != http.StatusOK {
		t.Errorf("o dono deveria ver a própria publicação: status %d", resposta.Code)
	}
}

func TestCadastroExigeAceiteDosTermos(t *testing.T) {
	db := testes.NovoBanco(t)
	base := `{"nome":"Ana","nick":"ana","email":"ana@teste.dev","senha":"demo1234"%s}`

	if resposta := chamar(t, 0, http.MethodPost, "/usuarios", fmt.Sprintf(base, "")); resposta.Code != http.StatusBadRequest {
		t.Fatalf("cadastro sem aceite: status %d", resposta.Code)
	}
	var total int
	db.QueryRow("select count(*) from usuarios").Scan(&total)
	if total != 0 {
		t.Fatal("o usuário foi criado sem aceitar os termos")
	}

	if resposta := chamar(t, 0, http.MethodPost, "/usuarios", fmt.Sprintf(base, `,"aceitouTermos":true`)); resposta.Code != http.StatusCreated {
		t.Fatalf("cadastro com aceite: status %d (%s)", resposta.Code, resposta.Body.String())
	}
	var aceitoEm sql.NullTime
	db.QueryRow("select termos_aceitos_em from usuarios").Scan(&aceitoEm)
	if !aceitoEm.Valid {
		t.Error("a data do aceite não foi registrada")
	}
}

func TestSilenciarEscondeDoFeedMasNaoDoPerfil(t *testing.T) {
	db := testes.NovoBanco(t)
	eu := testes.CriarUsuario(t, db, "eu", false)
	chato := testes.CriarUsuario(t, db, "chato", false)
	amigo := testes.CriarUsuario(t, db, "amigo", false)
	criarPublicacao(t, db, chato)
	criarPublicacao(t, db, amigo)

	total := func(caminho string) int {
		t.Helper()
		var lista []map[string]interface{}
		resposta := chamar(t, eu, http.MethodGet, caminho, "")
		if erro := json.NewDecoder(resposta.Body).Decode(&lista); erro != nil {
			t.Fatal(erro)
		}
		return len(lista)
	}
	feed := "/publicacoes?aba=todos"
	if n := total(feed); n != 2 {
		t.Fatalf("antes de silenciar: %d publicações", n)
	}

	if resposta := chamar(t, eu, http.MethodPost, fmt.Sprintf("/usuarios/%d/silenciar", chato), ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("silenciar: status %d", resposta.Code)
	}
	if n := total(feed); n != 1 {
		t.Errorf("depois de silenciar: %d publicações no feed", n)
	}
	if n := total(fmt.Sprintf("/usuarios/%d/publicacoes", chato)); n != 1 {
		t.Errorf("o perfil de quem foi silenciado deveria continuar visível: %d", n)
	}
	if resposta := chamar(t, eu, http.MethodPost, fmt.Sprintf("/usuarios/%d/silenciar", eu), ""); resposta.Code != http.StatusForbidden {
		t.Errorf("silenciar a si mesmo: status %d", resposta.Code)
	}

	chamar(t, eu, http.MethodPost, fmt.Sprintf("/usuarios/%d/dessilenciar", chato), "")
	if n := total(feed); n != 2 {
		t.Errorf("depois de reverter: %d publicações", n)
	}
}
