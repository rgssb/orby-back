package email

import (
	"fmt"
	"log"
	"mime"
	"net/smtp"
	"orby/api/src/config"
	"strings"
)

// EnviarRecuperacaoDeSenha manda o link de recuperação ao usuário. Sem SMTP configurado,
// o link é escrito no terminal da API (útil em desenvolvimento).
func EnviarRecuperacaoDeSenha(destinatario, nome, link string) error {
	if config.SMTPHost == "" {
		log.Printf("\n[RECUPERAÇÃO DE SENHA] SMTP não configurado. Link para %s (válido por 30 minutos):\n%s\n", destinatario, link)
		return nil
	}

	corpo := fmt.Sprintf(
		"Olá, %s!\r\n\r\nRecebemos um pedido para redefinir a sua senha no Orby.\r\nUse o link abaixo (válido por 30 minutos):\r\n\r\n%s\r\n\r\nSe não foi você, ignore este e-mail: a sua senha continua a mesma.\r\n",
		limparCabecalho(nome), link,
	)

	mensagem := strings.Join([]string{
		"From: Orby <" + limparCabecalho(config.SMTPRemetente) + ">",
		"To: " + limparCabecalho(destinatario),
		"Subject: " + mime.QEncoding.Encode("utf-8", "Recuperação de senha - Orby"),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		corpo,
	}, "\r\n")

	var autenticacao smtp.Auth
	if config.SMTPUsuario != "" {
		autenticacao = smtp.PlainAuth("", config.SMTPUsuario, config.SMTPSenha, config.SMTPHost)
	}

	return smtp.SendMail(
		config.SMTPHost+":"+config.SMTPPorta,
		autenticacao,
		config.SMTPRemetente,
		[]string{destinatario},
		[]byte(mensagem),
	)
}

// limparCabecalho impede injeção de cabeçalhos por quebras de linha
func limparCabecalho(valor string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(valor)
}
