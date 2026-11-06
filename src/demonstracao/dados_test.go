package demonstracao

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"orby/api/src/autenticacao"
	"orby/api/src/router"
	"orby/api/src/testes"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCriarDadosDeDemonstracao(t *testing.T) {
	db := testes.NovoBanco(t)
	perfil := testes.CriarUsuario(t, db, "perfil", false)
	if err := Criar(db, "senha-de-teste", []string{"perfil"}); err != nil {
		t.Fatal(err)
	}
	contar := func(consulta string, esperado int, args ...any) {
		t.Helper()
		var total int
		if err := db.QueryRow(consulta, args...).Scan(&total); err != nil {
			t.Fatal(err)
		}
		if total != esperado {
			t.Errorf("%s: obtido %d, esperado %d", consulta, total, esperado)
		}
	}
	contar("select count(*) from usuarios where nick like 'usuario\\_%' and admin = 0 and email_verificado_em is not null", 10)
	contar("select count(*) from seguidores where seguidor_id = ?", 4, perfil)
	contar("select count(*) from felas where dono_id = ?", 4, perfil)
	contar("select count(*) from publicacoes_felas", 1)
	contar("select count(*) from votos_enquetes", 1)
	contar("select count(*) from comentarios where parent_id is not null", 8)
	contar("select count(*) from reacoes_publicacoes", 8)
	contar("select count(*) from reacoes_comentarios", 8)
	contar("select count(*) from curtidas_publicacoes", 8)
	contar("select count(*) from publicacoes where curtidas = 1", 8)
	contar("select count(*) from pastas_publicacoes", 1)
	contar("select count(*) from reposts", 1)
	contar("select count(*) from grupo_membros where usuario_id = ?", 1, perfil)
	contar("select count(*) from conversas where usuario_menor = ? or usuario_maior = ?", 1, perfil, perfil)
	contar("select count(*) from denuncias", 1)
	contar("select count(*) from bloqueios", 1)
	var senha string
	if err := db.QueryRow("select senha from usuarios where nick = 'usuario_1'").Scan(&senha); err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(senha), []byte("senha-de-teste")); err != nil {
		t.Fatal(err)
	}
	token, err := autenticacao.CriarToken(perfil)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		caminho, texto string
	}{
		{"/publicacoes?aba=seguindo", "Café e ideias"},
		{"/publicacoes?aba=felas", "Para meus Felas"},
		{"/conversas", "usuario_1"},
		{"/grupos", "Papo de fim de tarde"},
		{"/notificacoes", "usuario_1"},
	} {
		req := httptest.NewRequest(http.MethodGet, caso.caminho, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.Gerar().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), caso.texto) {
			t.Errorf("%s: status %d, resposta %s", caso.caminho, rec.Code, rec.Body.String())
		}
	}
	if err := Criar(db, "senha-de-teste", []string{"perfil"}); err == nil {
		t.Error("não deve sobrescrever contas existentes")
	}
	contar("select count(*) from usuarios", 11)
}

func TestCriarFalhaSemAlterarDados(t *testing.T) {
	db := testes.NovoBanco(t)
	testes.CriarUsuario(t, db, "usuario_5", false)
	if err := Criar(db, "senha-de-teste", nil); err == nil {
		t.Fatal("conflito deveria cancelar a criação")
	}
	var total int
	if err := db.QueryRow("select count(*) from usuarios").Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("transação deixou %d contas", total)
	}
	if err := Criar(db, "senha-de-teste", []string{"ausente"}); err == nil || err == sql.ErrNoRows {
		t.Fatal("perfil ausente deveria ser informado")
	}
}
func TestAtualizarTextosAntigosPreservaEdicoes(t *testing.T) {
	db := testes.NovoBanco(t)
	if err := Criar(db, "senha-de-teste", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`update publicacoes p join usuarios u on u.id = p.autor_id
			set p.titulo = 'Publicação de exemplo 1', p.conteudo = 'Olá! Sou Usuario1. Esta publicação é um exemplo para conhecer o Orby. #exemplo'
			where u.nick = 'usuario_1' and p.titulo = 'Café e ideias'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`update publicacoes p join usuarios u on u.id = p.autor_id
			set p.titulo = 'Publicação de exemplo 2', p.conteudo = 'texto editado'
			where u.nick = 'usuario_2' and p.titulo = 'Fim de tarde'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`update comentarios c join publicacoes p on p.id = c.publicacao_id join usuarios u on u.id = p.autor_id
			set c.conteudo = 'Comentário de exemplo: gostei da publicação!'
			where u.nick = 'usuario_1' and c.parent_id is null`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`delete m from mensagens m join usuarios u on u.id = m.remetente_id where u.nick = 'usuario_2'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`update mensagens m join usuarios u on u.id = m.remetente_id
			set m.texto = 'Olá! Esta é uma mensagem de exemplo.' where u.nick = 'usuario_1'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`delete m from grupo_mensagens m join usuarios u on u.id = m.remetente_id where u.nick = 'usuario_1'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`update grupos set nome = 'Grupo de exemplo'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`update grupo_mensagens set texto = 'Olá! Este grupo é um exemplo para testar conversas.'`); err != nil {
		t.Fatal(err)
	}
	if err := AtualizarTextosAntigos(db); err != nil {
		t.Fatal(err)
	}
	if err := AtualizarTextosAntigos(db); err != nil {
		t.Fatal(err)
	}
	var titulo, conteudo, comentario string
	if err := db.QueryRow(`select p.titulo, p.conteudo, c.conteudo from publicacoes p
			join usuarios u on u.id = p.autor_id join comentarios c on c.publicacao_id = p.id and c.parent_id is null
			where u.nick = 'usuario_1' and p.titulo = ?`, assuntos[0].titulo).Scan(&titulo, &conteudo, &comentario); err != nil {
		t.Fatal(err)
	}
	if titulo != assuntos[0].titulo || conteudo != assuntos[0].conteudo || comentario != assuntos[0].comentario {
		t.Fatalf("textos não atualizados: %q, %q, %q", titulo, conteudo, comentario)
	}
	if err := db.QueryRow(`select p.conteudo from publicacoes p join usuarios u on u.id = p.autor_id
			where u.nick = 'usuario_2' and p.titulo = 'Publicação de exemplo 2'`).Scan(&conteudo); err != nil || conteudo != "texto editado" {
		t.Fatalf("edição não preservada: %q %v", conteudo, err)
	}
	var quantidade int
	if err := db.QueryRow("select count(*) from mensagens").Scan(&quantidade); err != nil || quantidade != 2 {
		t.Fatalf("conversa deve ter uma resposta: %d %v", quantidade, err)
	}
	if err := db.QueryRow("select count(*) from grupo_mensagens").Scan(&quantidade); err != nil || quantidade != 2 {
		t.Fatalf("grupo deve ter uma resposta: %d %v", quantidade, err)
	}
}
