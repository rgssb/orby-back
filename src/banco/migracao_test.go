package banco

import (
	"context"
	"orby/api/src/testes"
	"testing"
)

func TestMigrarCriaTudoEmBancoVazioERepete(t *testing.T) {
	db := testes.NovoBanco(t)

	conexao, erro := db.Conn(context.Background())
	if erro != nil {
		t.Fatal(erro)
	}
	defer conexao.Close()
	conexao.ExecContext(context.Background(), "set foreign_key_checks = 0")
	linhas, erro := conexao.QueryContext(context.Background(), "select table_name from information_schema.tables where table_schema = database()")
	if erro != nil {
		t.Fatal(erro)
	}
	var tabelas []string
	for linhas.Next() {
		var nome string
		linhas.Scan(&nome)
		tabelas = append(tabelas, nome)
	}
	linhas.Close()
	for _, nome := range tabelas {
		if _, erro = conexao.ExecContext(context.Background(), "drop table `"+nome+"`"); erro != nil {
			t.Fatal(erro)
		}
	}
	conexao.ExecContext(context.Background(), "set foreign_key_checks = 1")

	for i := 0; i < 2; i++ {
		if erro = Migrar(); erro != nil {
			t.Fatalf("Migrar (execução %d) em banco vazio: %v", i+1, erro)
		}
	}

	var usuarios int
	if erro = db.QueryRow("select count(*) from usuarios").Scan(&usuarios); erro != nil {
		t.Fatal(erro)
	}
}
