package banco

import (
	"orby/api/src/testes"
	"testing"
)

func TestMigrarAdicionaTituloSemPerderEnquetes(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	post, erro := db.Exec("insert into publicacoes (conteudo, autor_id) values ('conteúdo', ?)", autor)
	if erro != nil {
		t.Fatal(erro)
	}
	id, erro := post.LastInsertId()
	if erro != nil {
		t.Fatal(erro)
	}
	if _, erro = db.Exec("insert into enquetes (publicacao_id) values (?)", id); erro != nil {
		t.Fatal(erro)
	}
	if _, erro = db.Exec("alter table enquetes drop column titulo"); erro != nil {
		t.Fatal(erro)
	}
	for i := 0; i < 2; i++ {
		if erro = Migrar(); erro != nil {
			t.Fatalf("migração %d: %v", i+1, erro)
		}
	}
	var titulo string
	if erro = db.QueryRow("select titulo from enquetes where publicacao_id = ?", id).Scan(&titulo); erro != nil || titulo != "" {
		t.Fatalf("enquete existente não preservada: %q %v", titulo, erro)
	}
}
