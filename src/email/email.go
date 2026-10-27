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
	return enviar(destinatario, "Recuperação de senha - Orby", "RECUPERAÇÃO DE SENHA", "30 minutos", link, fmt.Sprintf(
		"Olá, %s!\r\n\r\nRecebemos um pedido para redefinir a sua senha no Orby.\r\nUse o link abaixo (válido por 30 minutos):\r\n\r\n%s\r\n\r\nSe não foi você, ignore este e-mail: a sua senha continua a mesma.\r\n",
		limparCabecalho(nome), link,
	))
}

// EnviarConfirmacaoDeEmail manda o link que confirma o e-mail da conta. Sem SMTP configurado,
// o link é escrito no terminal da API.
func EnviarConfirmacaoDeEmail(destinatario, nome, link string) error {
	return enviar(destinatario, "Confirme seu e-mail - Orby", "CONFIRMAÇÃO DE E-MAIL", "24 horas", link, fmt.Sprintf(
		"Olá, %s!\r\n\r\nBem-vindo ao Orby! Para ativar a sua conta, confirme este e-mail pelo link abaixo (válido por 24 horas):\r\n\r\n%s\r\n\r\nSe não foi você quem criou a conta, ignore este e-mail.\r\n",
		limparCabecalho(nome), link,
	))
}

func enviar(destinatario, assunto, rotuloDoLog, validade, link, corpo string) error {
	if config.SMTPHost == "" {
		log.Printf("\n[%s] SMTP não configurado. Link para %s (válido por %s):\n%s\n", rotuloDoLog, destinatario, validade, link)
		return nil
	}

	mensagem := strings.Join([]string{
		"From: Orby <" + limparCabecalho(config.SMTPRemetente) + ">",
		"To: " + limparCabecalho(destinatario),
		"Subject: " + mime.QEncoding.Encode("utf-8", assunto),
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
