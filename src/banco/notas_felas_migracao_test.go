package banco

import (
	"orby/api/src/testes"
	"testing"
)

func TestMigrarNotasFelasEmBancoExistente(t *testing.T) {
	db := testes.NovoBanco(t)
	id := testes.CriarUsuario(t, db, "pessoa", false)
	if _, err := db.Exec("drop table notas_felas"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("alter table usuarios drop column notas_felas_ativas"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := Migrar(); err != nil {
			t.Fatal(err)
		}
	}
	var ativas bool
	if err := db.QueryRow("select notas_felas_ativas from usuarios where id = ?", id).Scan(&ativas); err != nil || !ativas {
		t.Fatalf("notas devem iniciar ativas: %v %v", ativas, err)
	}
}
