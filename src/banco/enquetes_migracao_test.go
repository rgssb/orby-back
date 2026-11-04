package banco

import (
	"orby/api/src/testes"
	"testing"
)

func TestMigrarCriaEnquetesEmBancoExistente(t *testing.T) {
	db := testes.NovoBanco(t)
	for _, tabela := range []string{"votos_enquetes", "opcoes_enquetes", "enquetes"} {
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
