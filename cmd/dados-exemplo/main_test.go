package main

import (
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestBancoLocal(t *testing.T) {
	for _, caso := range []struct {
		net, addr, nome string
		permitido       bool
	}{
		{"tcp", "127.0.0.1:3306", "orby", true},
		{"tcp", "localhost:3306", "orby", true},
		{"tcp", "[::1]:3306", "orby", true},
		{"unix", "/tmp/mysql.sock", "orby", true},
		{"tcp", "192.0.2.1:3306", "orby", false},
		{"tcp", "db:3306", "orby", false},
		{"tcp", "localhost:3306", "", false},
	} {
		dsn := &mysql.Config{Net: caso.net, Addr: caso.addr, DBName: caso.nome}
		if got := bancoLocal(dsn); got != caso.permitido {
			t.Errorf("%s %s %q: obtido %v", caso.net, caso.addr, caso.nome, got)
		}
	}
}
