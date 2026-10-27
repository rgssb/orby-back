package controllers_test

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"orby/api/src/autenticacao"
	"orby/api/src/repositorios"
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
		{http.MethodGet, "/publicacoes/" + id + "/imagens/1", ""},
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

func TestLoginExigeEmailConfirmado(t *testing.T) {
	db := testes.NovoBanco(t)
	cadastro := `{"nome":"Ana","nick":"ana","email":"ana@teste.dev","senha":"demo1234","aceitouTermos":true}`
	login := `{"email":"ana@teste.dev","senha":"demo1234"}`
	if resposta := chamar(t, 0, http.MethodPost, "/usuarios", cadastro); resposta.Code != http.StatusCreated {
		t.Fatalf("cadastro: status %d", resposta.Code)
	}
	if resposta := chamar(t, 0, http.MethodPost, "/login", login); resposta.Code != http.StatusForbidden {
		t.Fatalf("login sem confirmar: status %d", resposta.Code)
	}

	guardar := func(token string, validade string) {
		hash := sha256.Sum256([]byte(token))
		db.Exec("delete from verificacoes_email")
		db.Exec("insert into verificacoes_email (usuario_id, token_hash, expiraEm) select id, ?, "+validade+" from usuarios", hex.EncodeToString(hash[:]))
	}
	guardar("vencido", "date_sub(now(), interval 1 hour)")
	if resposta := chamar(t, 0, http.MethodPost, "/confirmar-email", `{"token":"vencido"}`); resposta.Code != http.StatusBadRequest {
		t.Errorf("token expirado: status %d", resposta.Code)
	}
	if resposta := chamar(t, 0, http.MethodPost, "/confirmar-email", `{"token":"inexistente"}`); resposta.Code != http.StatusBadRequest {
		t.Errorf("token inexistente: status %d", resposta.Code)
	}
	if resposta := chamar(t, 0, http.MethodPost, "/login", login); resposta.Code != http.StatusForbidden {
		t.Fatal("um token inválido não pode liberar o login")
	}

	guardar("valido", "date_add(now(), interval 1 hour)")
	if resposta := chamar(t, 0, http.MethodPost, "/confirmar-email", `{"token":"valido"}`); resposta.Code != http.StatusNoContent {
		t.Fatalf("confirmar: status %d", resposta.Code)
	}
	if resposta := chamar(t, 0, http.MethodPost, "/login", login); resposta.Code != http.StatusOK {
		t.Fatalf("login depois de confirmar: status %d", resposta.Code)
	}
	if resposta := chamar(t, 0, http.MethodPost, "/confirmar-email", `{"token":"valido"}`); resposta.Code != http.StatusBadRequest {
		t.Errorf("o link só pode ser usado uma vez: status %d", resposta.Code)
	}
}

func TestReenviarConfirmacaoNaoRevelaContas(t *testing.T) {
	testes.NovoBanco(t)
	for _, email := range []string{"ninguem@teste.dev", "tambem-ninguem@teste.dev"} {
		resposta := chamar(t, 0, http.MethodPost, "/reenviar-confirmacao", fmt.Sprintf(`{"email":%q}`, email))
		if resposta.Code != http.StatusOK {
			t.Errorf("reenviar para e-mail desconhecido: status %d", resposta.Code)
		}
	}
}

func TestRespostasDeComentariosAninhadas(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	bia := testes.CriarUsuario(t, db, "bia", false)
	caio := testes.CriarUsuario(t, db, "caio", false)
	publicacao := criarPublicacao(t, db, autor)
	outra := criarPublicacao(t, db, autor)

	comentar := func(usuario uint64, pub, corpo string) (int, uint64) {
		resposta := chamar(t, usuario, http.MethodPost, "/publicacoes/"+pub+"/comentarios", corpo)
		var criado struct {
			ID uint64 `json:"id"`
		}
		json.NewDecoder(resposta.Body).Decode(&criado)
		return resposta.Code, criado.ID
	}
	_, principal := comentar(bia, publicacao, `{"conteudo":"comentário principal"}`)
	if codigo, _ := comentar(caio, publicacao, fmt.Sprintf(`{"conteudo":"resposta","parentId":%d}`, principal)); codigo != http.StatusCreated {
		t.Fatalf("responder: status %d", codigo)
	}
	var respostaDoCaio uint64
	db.QueryRow("select id from comentarios where autor_id = ? and parent_id is not null", caio).Scan(&respostaDoCaio)
	// responder a uma resposta liga ao comentário respondido
	comentar(autor, publicacao, fmt.Sprintf(`{"conteudo":"resposta da resposta","parentId":%d}`, respostaDoCaio))
	var ligadas int
	db.QueryRow("select count(*) from comentarios where parent_id = ?", respostaDoCaio).Scan(&ligadas)
	if ligadas != 1 {
		t.Errorf("respostas ligadas à resposta: %d, esperado 1", ligadas)
	}

	// passando da profundidade máxima, a resposta sobe um nível
	pai := respostaDoCaio
	for i := 0; i < 3; i++ {
		_, pai = comentar(bia, publicacao, fmt.Sprintf(`{"conteudo":"fundo","parentId":%d}`, pai))
	}
	var profundidade int
	for atual := pai; atual != 0; profundidade++ {
		db.QueryRow("select coalesce(parent_id, 0) from comentarios where id = ?", atual).Scan(&atual)
	}
	if profundidade != repositorios.ProfundidadeMaximaDeRespostas+1 {
		t.Errorf("profundidade da cadeia: %d", profundidade)
	}

	if codigo, _ := comentar(caio, outra, fmt.Sprintf(`{"conteudo":"x","parentId":%d}`, principal)); codigo != http.StatusBadRequest {
		t.Errorf("responder comentário de outra publicação: status %d", codigo)
	}
	if codigo, _ := comentar(caio, publicacao, `{"conteudo":"x","parentId":99999}`); codigo != http.StatusBadRequest {
		t.Errorf("responder comentário inexistente: status %d", codigo)
	}

	var principais []struct {
		ID             uint64 `json:"id"`
		Respostas      uint64 `json:"respostas"`
		TotalRespostas uint64 `json:"totalRespostas"`
	}
	json.NewDecoder(chamar(t, caio, http.MethodGet, "/publicacoes/"+publicacao+"/comentarios", "").Body).Decode(&principais)
	if len(principais) != 1 || principais[0].Respostas != 1 || principais[0].TotalRespostas != 5 {
		t.Fatalf("a lista principal deve ter só o comentário principal com 2 respostas: %+v", principais)
	}
	var respostas []map[string]interface{}
	json.NewDecoder(chamar(t, caio, http.MethodGet, fmt.Sprintf("/comentarios/%d/respostas", principal), "").Body).Decode(&respostas)
	if len(respostas) != 1 {
		t.Errorf("respostas listadas: %d", len(respostas))
	}

	var avisosParaBia, avisosParaOAutor int
	db.QueryRow("select count(*) from notificacoes where tipo = 'resposta' and usuario_id = ?", bia).Scan(&avisosParaBia)
	db.QueryRow("select count(*) from notificacoes where tipo = 'resposta' and usuario_id = ?", autor).Scan(&avisosParaOAutor)
	if avisosParaBia != 1 || avisosParaOAutor != 0 {
		t.Errorf("avisos de resposta: bia=%d (esperado 1), dono da publicação=%d (esperado 0)", avisosParaBia, avisosParaOAutor)
	}

	if r := chamar(t, bia, http.MethodDelete, fmt.Sprintf("/comentarios/%d", principal), ""); r.Code >= 300 {
		t.Errorf("delete: %d %s", r.Code, r.Body.String())
	}
	var restantes int
	db.QueryRow("select count(*) from comentarios").Scan(&restantes)
	if restantes != 0 {
		t.Errorf("apagar o comentário principal deve apagar as respostas: restaram %d", restantes)
	}
}

func TestApagarPublicacaoComConversaFunda(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	publicacao := criarPublicacao(t, db, autor)

	var pai interface{}
	for i := 0; i < repositorios.ProfundidadeMaximaDeRespostas+1; i++ {
		resultado, erro := db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo, parent_id) values (?, ?, 'x', ?)", publicacao, autor, pai)
		if erro != nil {
			t.Fatal(erro)
		}
		pai, _ = resultado.LastInsertId()
	}

	if resposta := chamar(t, autor, http.MethodDelete, "/publicacoes/"+publicacao, ""); resposta.Code >= 300 {
		t.Fatalf("apagar publicação: %d %s", resposta.Code, resposta.Body.String())
	}
}

func TestRespostasRespeitamPerfilPrivado(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	estranho := testes.CriarUsuario(t, db, "estranho", false)
	publicacao := criarPublicacao(t, db, dono)
	resultado, _ := db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'c')", publicacao, dono)
	id, _ := resultado.LastInsertId()

	if resposta := chamar(t, estranho, http.MethodGet, fmt.Sprintf("/comentarios/%d/respostas", id), ""); resposta.Code != http.StatusNotFound {
		t.Errorf("respostas de publicação privada: status %d", resposta.Code)
	}
}

func TestRepostarPublicacoes(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	bia := testes.CriarUsuario(t, db, "bia", false)
	caio := testes.CriarUsuario(t, db, "caio", false)
	original := criarPublicacao(t, db, autor)
	caminho := "/publicacoes/" + original + "/repostar"

	if codigo := chamar(t, bia, http.MethodPost, caminho, `{}`).Code; codigo != http.StatusCreated {
		t.Fatalf("repostar: status %d", codigo)
	}
	if codigo := chamar(t, bia, http.MethodPost, caminho, `{}`).Code; codigo != http.StatusConflict {
		t.Errorf("repost simples duplicado: status %d", codigo)
	}
	if codigo := chamar(t, bia, http.MethodPost, caminho, `{"conteudo":"concordo"}`).Code; codigo != http.StatusCreated {
		t.Errorf("citação: status %d", codigo)
	}

	// repostar um repost aponta para a publicação de origem
	var repostDaBia string
	db.QueryRow("select p.id from publicacoes p where p.autor_id = ? and p.conteudo = ''", bia).Scan(&repostDaBia)
	chamar(t, caio, http.MethodPost, "/publicacoes/"+repostDaBia+"/repostar", `{}`)
	var apontando int
	db.QueryRow("select count(*) from reposts where original_id = ?", original).Scan(&apontando)
	if apontando != 3 {
		t.Errorf("reposts ligados ao original: %d, esperado 3", apontando)
	}

	var avisos int
	db.QueryRow("select count(*) from notificacoes where tipo = 'repost' and usuario_id = ?", autor).Scan(&avisos)
	if avisos != 3 {
		t.Errorf("avisos de repost: %d, esperado 3", avisos)
	}

	var feed []struct {
		Reposts             uint64 `json:"reposts"`
		RepostadoPorUsuario bool   `json:"repostadoPorUsuario"`
		RepostDe            *struct {
			ID           uint64 `json:"id"`
			Conteudo     string `json:"conteudo"`
			Indisponivel bool   `json:"indisponivel"`
		} `json:"repostDe"`
	}
	json.NewDecoder(chamar(t, bia, http.MethodGet, fmt.Sprintf("/usuarios/%d/publicacoes", bia), "").Body).Decode(&feed)
	if len(feed) != 2 || feed[0].RepostDe == nil || feed[0].RepostDe.Conteudo != "conteudo secreto" {
		t.Fatalf("o repost deve trazer o original: %+v", feed)
	}

	// desfazer remove só o repost sem comentário
	if codigo := chamar(t, bia, http.MethodDelete, caminho, "").Code; codigo != http.StatusNoContent {
		t.Errorf("desfazer repost: status %d", codigo)
	}
	db.QueryRow("select count(*) from publicacoes where autor_id = ?", bia).Scan(&apontando)
	if apontando != 1 {
		t.Errorf("publicações da bia depois de desfazer: %d, esperado 1 (a citação)", apontando)
	}

	// se o autor virar privado, quem não o segue vê o original como indisponível
	db.Exec("update usuarios set privado = 1 where id = ?", autor)
	feed = nil
	json.NewDecoder(chamar(t, caio, http.MethodGet, fmt.Sprintf("/usuarios/%d/publicacoes", caio), "").Body).Decode(&feed)
	if len(feed) != 1 || feed[0].RepostDe == nil || !feed[0].RepostDe.Indisponivel || feed[0].RepostDe.Conteudo != "" {
		t.Errorf("original de conta privada deve aparecer indisponível: %+v", feed)
	}
	if codigo := chamar(t, caio, http.MethodPost, caminho, `{}`).Code; codigo != http.StatusNotFound {
		t.Errorf("repostar publicação de conta privada: status %d", codigo)
	}

	// apagar o original apaga os reposts
	db.Exec("update usuarios set privado = 0 where id = ?", autor)
	chamar(t, autor, http.MethodDelete, "/publicacoes/"+original, "")
	var restantes int
	db.QueryRow("select count(*) from publicacoes").Scan(&restantes)
	if restantes != 0 {
		t.Errorf("publicações restantes depois de apagar o original: %d", restantes)
	}
}

func TestGruposViaAPI(t *testing.T) {
	db := testes.NovoBanco(t)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	caio := testes.CriarUsuario(t, db, "caio", false)

	r := chamar(t, ana, http.MethodPost, "/grupos", fmt.Sprintf(`{"nome":"Amigos","membros":[%d]}`, beto))
	if r.Code != http.StatusCreated {
		t.Fatalf("criar: %d %s", r.Code, r.Body)
	}
	var criado struct{ ID uint64 }
	json.Unmarshal(r.Body.Bytes(), &criado)
	base := fmt.Sprintf("/grupos/%d", criado.ID)

	if r = chamar(t, ana, http.MethodPost, "/grupos", `{"nome":"  ","membros":[1]}`); r.Code != http.StatusBadRequest {
		t.Fatalf("nome em branco: %d", r.Code)
	}
	if r = chamar(t, caio, http.MethodGet, base, ""); r.Code != http.StatusNotFound {
		t.Fatalf("não membro não vê o grupo: %d", r.Code)
	}
	if r = chamar(t, beto, http.MethodPost, base+"/mensagens", `{"texto":"oi"}`); r.Code != http.StatusCreated {
		t.Fatalf("enviar: %d %s", r.Code, r.Body)
	}
	if r = chamar(t, beto, http.MethodPost, base+"/membros", fmt.Sprintf(`{"usuarioId":%d}`, caio)); r.Code != http.StatusForbidden {
		t.Fatalf("sem permissão: %d", r.Code)
	}
	if r = chamar(t, ana, http.MethodPost, base+"/membros", fmt.Sprintf(`{"usuarioId":%d}`, caio)); r.Code != http.StatusNoContent {
		t.Fatalf("adicionar: %d %s", r.Code, r.Body)
	}
}
