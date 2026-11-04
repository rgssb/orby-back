package banco

import (
	"orby/api/src/testes"
	"testing"
)

func TestMigrarCriaPastasEmBancoExistente(t *testing.T) {
	db := testes.NovoBanco(t)
	for _, tabela := range []string{"pastas_publicacoes", "pastas_salvos"} {
		if _, erro := db.Exec("drop table " + tabela); erro != nil {
			t.Fatal(erro)
		}
	}
	for i := 0; i < 2; i++ {
		if erro := Migrar(); erro != nil {
			t.Fatalf("migração %d: %v", i+1, erro)
		}
	}
}
