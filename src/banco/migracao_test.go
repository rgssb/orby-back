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

func TestMigrarMarcaContasAntigasComoConfirmadas(t *testing.T) {
	db := testes.NovoBanco(t)
	testes.CriarUsuario(t, db, "antigo", false)
	db.Exec("update usuarios set email_verificado_em = null")
	db.Exec("alter table usuarios drop column email_verificado_em")

	if erro := Migrar(); erro != nil {
		t.Fatal(erro)
	}

	var confirmado bool
	db.QueryRow("select email_verificado_em is not null from usuarios").Scan(&confirmado)
	if !confirmado {
		t.Error("conta existente deveria ficar com o e-mail confirmado")
	}

	// uma conta nova (depois da migração) nasce sem confirmação e não é afetada por novas migrações
	db.Exec("insert into usuarios (nome, nick, email, senha) values ('n', 'novo', 'novo@orby.test', 'x')")
	if erro := Migrar(); erro != nil {
		t.Fatal(erro)
	}
	db.QueryRow("select email_verificado_em is not null from usuarios where nick = 'novo'").Scan(&confirmado)
	if confirmado {
		t.Error("migrar de novo não pode confirmar contas novas")
	}
}

func TestMigrarPermiteNovoCicloDeDenunciaSemPerderHistorico(t *testing.T) {
	db := testes.NovoBanco(t)
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	pessoa := testes.CriarUsuario(t, db, "pessoa", false)
	if _, erro := db.Exec(`alter table denuncias add unique key uq_denuncia (denunciante_id, tipo, alvo_id),
		drop index uq_denuncia_aberta, drop column denuncia_aberta`); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := db.Exec("insert into denuncias (denunciante_id, dono_id, tipo, alvo_id, motivo, resolvida) values (?, ?, 'perfil', ?, 'spam', 1)", pessoa, alvo, alvo); erro != nil {
		t.Fatal(erro)
	}
	for i := 0; i < 2; i++ {
		if erro := Migrar(); erro != nil {
			t.Fatalf("migração %d: %v", i+1, erro)
		}
	}
	inserir := func() error {
		_, erro := db.Exec("insert into denuncias (denunciante_id, dono_id, tipo, alvo_id, motivo) values (?, ?, 'perfil', ?, 'spam')", pessoa, alvo, alvo)
		return erro
	}
	if erro := inserir(); erro != nil {
		t.Fatalf("denúncia arquivada deve permitir nova: %v", erro)
	}
	if erro := inserir(); erro == nil {
		t.Fatal("duas denúncias abertas para o mesmo alvo devem falhar")
	}
	if _, erro := db.Exec("update denuncias set resolvida = 1 where denunciante_id = ? and alvo_id = ?", pessoa, alvo); erro != nil {
		t.Fatal(erro)
	}
	if erro := inserir(); erro != nil {
		t.Fatalf("deve permitir mais de um ciclo de arquivamento: %v", erro)
	}
	var quantidade int
	if erro := db.QueryRow("select count(*) from denuncias").Scan(&quantidade); erro != nil || quantidade != 3 {
		t.Fatalf("histórico não foi preservado: quantidade=%d erro=%v", quantidade, erro)
	}
}
