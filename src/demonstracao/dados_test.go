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
		{"/publicacoes?aba=seguindo", "Publicação de exemplo 1"},
		{"/publicacoes?aba=felas", "Para meus Felas"},
		{"/conversas", "usuario_1"},
		{"/grupos", "Grupo de exemplo"},
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
