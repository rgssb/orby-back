package modelos

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// LimiteDeMensagem é o tamanho máximo de uma mensagem em caracteres
const LimiteDeMensagem = 1000

// Mensagem é uma mensagem direta dentro de uma conversa
type Mensagem struct {
	ID       uint64    `json:"id"`
	Texto    string    `json:"texto"`
	Minha    bool      `json:"minha"`
	Lida     bool      `json:"lida"`
	CriadoEm time.Time `json:"criadoEm"`
}

// Conversa resume a troca de mensagens entre o usuário autenticado e outra pessoa.
// Solicitacao indica um pedido de conversa recebido (a outra pessoa não é seguida de volta)
// e Aguardando indica que a pessoa ainda não aceitou uma conversa iniciada por mim.
type Conversa struct {
	UsuarioID      uint64    `json:"usuarioId"`
	Nome           string    `json:"nome"`
	Nick           string    `json:"nick"`
	FotoVersao     uint64    `json:"fotoVersao"`
	UltimaMensagem string    `json:"ultimaMensagem"`
	UltimaMinha    bool      `json:"ultimaMinha"`
	AtualizadoEm   time.Time `json:"atualizadoEm"`
	NaoLidas       uint64    `json:"naoLidas"`
	Solicitacao    bool      `json:"solicitacao"`
	Aguardando     bool      `json:"aguardando"`
	// Bloqueei: bloqueei esta pessoa. Bloqueado: ela me bloqueou. Desculpa: recebi um pedido de desculpas dela
	Bloqueei  bool `json:"bloqueei"`
	Bloqueado bool `json:"bloqueado"`
	Desculpa  bool `json:"desculpa"`
}

// PrepararTexto valida e formata o texto de uma mensagem
func PrepararTexto(texto string) (string, error) {
	texto = strings.TrimSpace(texto)
	if texto == "" {
		return "", errors.New("A mensagem não pode estar em branco")
	}
	if utf8.RuneCountInString(texto) > LimiteDeMensagem {
		return "", errors.New("A mensagem deve ter no máximo 1000 caracteres")
	}
	return texto, nil
}

// Dialogo é a conversa aberta com uma pessoa: o cabeçalho e as mensagens trocadas
type Dialogo struct {
	Conversa
	Mensagens []Mensagem `json:"mensagens"`
}
