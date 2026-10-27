// Package testes reúne utilitários para testes que precisam de um MySQL real.
package testes

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"orby/api/src/config"
	"orby/api/src/midia"
)

// NovoBanco devolve uma conexão com um banco recém-criado a partir de sql/sql.sql.
// Usa ORBY_TEST_DSN (ex.: usuario:senha@/orby_test?parseTime=True) e é pulado quando ela não existe.
// O nome do banco precisa terminar em "_test" porque as tabelas são recriadas a cada chamada.
func NovoBanco(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("ORBY_TEST_DSN")
	if dsn == "" {
		t.Skip("ORBY_TEST_DSN não definido; teste de banco ignorado")
	}
	cfg, erro := mysql.ParseDSN(dsn)
	if erro != nil {
		t.Fatalf("ORBY_TEST_DSN inválido: %v", erro)
	}
	if !strings.HasSuffix(cfg.DBName, "_test") {
		t.Fatalf("recusando rodar no banco %q: o nome precisa terminar em _test", cfg.DBName)
	}
	cfg.MultiStatements = true
	cfg.ParseTime = true

	db, erro := sql.Open("mysql", cfg.FormatDSN())
	if erro != nil {
		t.Fatal(erro)
	}
	t.Cleanup(func() { db.Close() })

	// Pacotes de teste rodam em paralelo e dividem o mesmo banco; o lock serializa os testes
	trava, erro := db.Conn(context.Background())
	if erro != nil {
		t.Fatal(erro)
	}
	var obtido int
	if erro = trava.QueryRowContext(context.Background(), "select get_lock('orby_testes', 120)").Scan(&obtido); erro != nil || obtido != 1 {
		t.Fatalf("não foi possível obter o lock dos testes: %v", erro)
	}
	t.Cleanup(func() {
		trava.ExecContext(context.Background(), "select release_lock('orby_testes')")
		trava.Close()
	})
	midia.Dir = t.TempDir()
	config.StringConexaoBanco = cfg.FormatDSN()
	config.SecretKey = []byte("segredo-de-teste")

	if _, erro = db.Exec(esquema(t)); erro != nil {
		t.Fatalf("não foi possível criar o esquema: %v", erro)
	}
	return db
}

// esquema lê sql/sql.sql sem as linhas que criam e selecionam o banco "orby"
func esquema(t *testing.T) string {
	t.Helper()
	_, arquivo, _, _ := runtime.Caller(0)
	conteudo, erro := os.ReadFile(filepath.Join(filepath.Dir(arquivo), "..", "..", "sql", "sql.sql"))
	if erro != nil {
		t.Fatal(erro)
	}

	var linhas []string
	for _, linha := range strings.Split(string(conteudo), "\n") {
		maiuscula := strings.ToUpper(strings.TrimSpace(linha))
		if strings.HasPrefix(maiuscula, "CREATE DATABASE") || strings.HasPrefix(maiuscula, "USE ") {
			continue
		}
		linhas = append(linhas, linha)
	}
	return strings.Join(linhas, "\n")
}

// CriarUsuario insere um usuário e devolve o ID
func CriarUsuario(t *testing.T, db *sql.DB, nick string, privado bool) uint64 {
	t.Helper()
	resultado, erro := db.Exec(
		"insert into usuarios (nome, nick, email, senha, privado, email_verificado_em) values (?, ?, ?, 'x', ?, now())",
		nick, nick, nick+"@orby.test", privado,
	)
	if erro != nil {
		t.Fatal(erro)
	}
	id, _ := resultado.LastInsertId()
	return uint64(id)
}
