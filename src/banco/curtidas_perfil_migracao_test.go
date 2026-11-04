package banco

import (
	"orby/api/src/testes"
	"testing"
)

func TestMigrarPrivacidadeDeCurtidasEmBancoExistente(t *testing.T) {
	db := testes.NovoBanco(t)
	id := testes.CriarUsuario(t, db, "pessoa", false)
	if _, erro := db.Exec("alter table usuarios drop column curtidas_publicas"); erro != nil {
		t.Fatal(erro)
	}
	for i := 0; i < 2; i++ {
		if erro := Migrar(); erro != nil {
			t.Fatalf("migração %d: %v", i+1, erro)
		}
	}
	var publica bool
	if erro := db.QueryRow("select curtidas_publicas from usuarios where id = ?", id).Scan(&publica); erro != nil || publica {
		t.Fatalf("lista deve iniciar privada: %v %v", publica, erro)
	}
}
