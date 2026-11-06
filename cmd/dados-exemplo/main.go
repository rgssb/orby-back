// Comando manual para popular um banco local com contas fictícias.
package main

import (
	"flag"
	"fmt"
	"net"
	"orby/api/src/banco"
	"orby/api/src/config"
	"orby/api/src/demonstracao"
	"os"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func main() {
	executar := flag.Bool("executar", false, "confirma a criação de dados fictícios no banco local")
	perfis := flag.String("perfis", "", "nicks existentes a conectar, separados por vírgula")
	flag.Parse()
	if !*executar {
		fmt.Fprintln(os.Stderr, "Use -executar para confirmar; consulte o README antes de usar.")
		os.Exit(1)
	}
	senha := os.Getenv("ORBY_DEMO_SENHA")
	if senha == "" {
		fmt.Fprintln(os.Stderr, "Defina ORBY_DEMO_SENHA para as contas fictícias.")
		os.Exit(1)
	}
	config.Carregar()
	dsn, err := mysql.ParseDSN(config.StringConexaoBanco)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Configuração de banco inválida.")
		os.Exit(1)
	}
	if !bancoLocal(dsn) {
		fmt.Fprintln(os.Stderr, "O gerador só pode ser usado em um banco local.")
		os.Exit(1)
	}
	db, err := banco.Conectar()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Não foi possível conectar ao banco local.")
		os.Exit(1)
	}
	defer db.Close()
	var nicks []string
	if strings.TrimSpace(*perfis) != "" {
		for _, perfil := range strings.Split(*perfis, ",") {
			perfil = strings.TrimSpace(perfil)
			if perfil == "" {
				fmt.Fprintln(os.Stderr, "A lista de perfis contém um nick vazio.")
				os.Exit(1)
			}
			nicks = append(nicks, perfil)
		}
	}
	if err = demonstracao.Criar(db, senha, nicks); err != nil {
		fmt.Fprintln(os.Stderr, "Não foi possível criar os dados de exemplo:", err)
		os.Exit(1)
	}
	fmt.Println("Criadas 10 contas fictícias e interações de demonstração.")
}

func bancoLocal(dsn *mysql.Config) bool {
	if dsn.DBName == "" {
		return false
	}
	if dsn.Net == "unix" {
		return true
	}
	if dsn.Net != "tcp" {
		return false
	}
	host, _, err := net.SplitHostPort(dsn.Addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return strings.EqualFold(host, "localhost") || (ip != nil && ip.IsLoopback())
}
