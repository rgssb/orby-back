package modelos

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Limites dos grupos de conversa
const (
	MaximoDeMembros = 67
	LimiteNomeGrupo = 60
)

// PrepararNomeDoGrupo valida e formata o nome de um grupo
func PrepararNomeDoGrupo(nome string) (string, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", errors.New("O grupo precisa de um nome")
	}
	if utf8.RuneCountInString(nome) > LimiteNomeGrupo {
		return "", errors.New("O nome do grupo deve ter no máximo 60 caracteres")
	}
	return nome, nil
}

// MembroDeGrupo é uma pessoa dentro de um grupo
type MembroDeGrupo struct {
	ID            uint64 `json:"id"`
	Nome          string `json:"nome"`
	Nick          string `json:"nick"`
	FotoVersao    uint64 `json:"fotoVersao"`
	Criador       bool   `json:"criador"`
	PodeAdicionar bool   `json:"podeAdicionar"`
}

// MensagemDeGrupo é uma mensagem enviada ao grupo
type MensagemDeGrupo struct {
	ID          uint64    `json:"id"`
	RemetenteID uint64    `json:"remetenteId"`
	Nome        string    `json:"nome"`
	Nick        string    `json:"nick"`
	Texto       string    `json:"texto"`
	Minha       bool      `json:"minha"`
	CriadoEm    time.Time `json:"criadoEm"`
}

// Grupo é o resumo de um grupo na lista de conversas
type Grupo struct {
	ID             uint64     `json:"id"`
	Nome           string     `json:"nome"`
	FotoVersao     uint64     `json:"fotoVersao"`
	Membros        uint64     `json:"membros"`
	NaoLidas       uint64     `json:"naoLidas"`
	UltimaMensagem string     `json:"ultimaMensagem"`
	UltimaEm       *time.Time `json:"ultimaEm"`
}

// DialogoDeGrupo é o grupo aberto: cabeçalho, membros e mensagens
type DialogoDeGrupo struct {
	Grupo
	EuID             uint64            `json:"euId"`
	EuCriador        bool              `json:"euCriador"`
	EuPossoAdicionar bool              `json:"euPossoAdicionar"`
	ListaDeMembros   []MembroDeGrupo   `json:"listaDeMembros"`
	MensagensDoGrupo []MensagemDeGrupo `json:"mensagensDoGrupo"`
}

// PessoaParaGrupo é alguém que o usuário pode colocar num grupo
type PessoaParaGrupo struct {
	ID         uint64 `json:"id"`
	Nome       string `json:"nome"`
	Nick       string `json:"nick"`
	FotoVersao uint64 `json:"fotoVersao"`
	Mutuo      bool   `json:"mutuo"`
}
