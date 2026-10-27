package modelos

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Comentario representa um comentário feito em uma publicação.
type Comentario struct {
	ID                uint64    `json:"id,omitempty"`
	PublicacaoID      uint64    `json:"publicacaoId,omitempty"`
	AutorID           uint64    `json:"autorId,omitempty"`
	AutorNick         string    `json:"autorNick,omitempty"`
	AutorFotoVersao   uint64    `json:"autorFotoVersao"`
	Conteudo          string    `json:"conteudo,omitempty"`
	CriadoEm          time.Time `json:"criadoEm,omitempty"`
	Editado           bool      `json:"editado"`
	Curtidas          uint64    `json:"curtidas"`
	CurtidoPorUsuario bool      `json:"curtidoPorUsuario"`
	ParentID          uint64    `json:"parentId,omitempty"`
	Respostas         uint64    `json:"respostas"`
}

// Preparar valida e formata o comentário recebido
func (comentario *Comentario) Preparar() error {
	comentario.Conteudo = strings.TrimSpace(comentario.Conteudo)

	if comentario.Conteudo == "" {
		return errors.New("O comentário não pode estar em branco")
	}

	if utf8.RuneCountInString(comentario.Conteudo) > 500 {
		return errors.New("O comentário deve ter no máximo 500 caracteres")
	}

	return nil
}
