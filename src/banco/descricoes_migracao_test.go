package banco

import (
	"orby/api/src/testes"
	"testing"
)

func TestMigrarAdicionaDescricoesEmImagensAntigas(t *testing.T) {
	db := testes.NovoBanco(t)
	for _, tabela := range []string{"imagens_publicacoes", "imagens_extras_publicacoes"} {
		if _, erro := db.Exec("alter table " + tabela + " drop column descricao"); erro != nil {
			t.Fatal(erro)
		}
	}
	for i := 0; i < 2; i++ {
		if erro := Migrar(); erro != nil {
			t.Fatalf("migração %d: %v", i+1, erro)
		}
	}
	for _, tabela := range []string{"imagens_publicacoes", "imagens_extras_publicacoes"} {
		var quantidade int
		if erro := db.QueryRow("select count(*) from information_schema.columns where table_schema = database() and table_name = ? and column_name = 'descricao'", tabela).Scan(&quantidade); erro != nil || quantidade != 1 {
			t.Fatalf("coluna de %s não criada: %d %v", tabela, quantidade, erro)
		}
	}
}
