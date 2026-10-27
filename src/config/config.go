package config

import (
	"fmt"
	"log"
	"orby/api/src/midia"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	//StringConexaoBanco é a string de conexao com o Mysql
	StringConexaoBanco = ""
	//Porta é onde a API vai estar rodando
	Porta = 0

	//SecretKey é a chave que vai ser usada para assinar o token
	SecretKey []byte

	//FrontURL é o endereço público da aplicação web, usado nos links enviados por e-mail
	FrontURL = "http://localhost:3000"

	//SMTPHost, SMTPPorta, SMTPUsuario, SMTPSenha e SMTPRemetente configuram o envio de e-mail.
	//Sem SMTPHost, o link de recuperação de senha é apenas escrito no terminal.
	SMTPHost      = ""
	SMTPPorta     = "587"
	SMTPUsuario   = ""
	SMTPSenha     = ""
	SMTPRemetente = ""
)

// Carregar vai inicializar as variaveis de ambiente
func Carregar() {
	var erro error

	if erro = godotenv.Load(); erro != nil {
		log.Fatal(erro)
	}

	Porta, erro = strconv.Atoi(os.Getenv("API_PORT"))
	if erro != nil {
		Porta = 9000
	}

	StringConexaoBanco = fmt.Sprintf("%s:%s@/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USUARIO"),
		os.Getenv("DB_SENHA"),
		os.Getenv("DB_NOME"),
	)

	SecretKey = []byte(os.Getenv("SECRET_KEY"))

	if valor := os.Getenv("MEDIA_DIR"); valor != "" {
		midia.Dir = valor
	}

	if valor := os.Getenv("FRONT_URL"); valor != "" {
		FrontURL = strings.TrimRight(valor, "/")
	}
	SMTPHost = os.Getenv("SMTP_HOST")
	if valor := os.Getenv("SMTP_PORTA"); valor != "" {
		SMTPPorta = valor
	}
	SMTPUsuario = os.Getenv("SMTP_USUARIO")
	SMTPSenha = os.Getenv("SMTP_SENHA")
	SMTPRemetente = os.Getenv("SMTP_REMETENTE")
	if SMTPRemetente == "" {
		SMTPRemetente = SMTPUsuario
	}
}
