package modelos

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Publicacao representa uma publicação feita por um usuário.
type Publicacao struct {
	ID                  uint64              `json:"id,omitempty"`
	Titulo              string              `json:"titulo,omitempty"`
	Conteudo            string              `json:"conteudo,omitempty"`
	AutorID             uint64              `json:"autorId,omitempty"`
	AutorNick           string              `json:"autorNick,omitempty"`
	AutorFotoVersao     uint64              `json:"autorFotoVersao"`
	Curtidas            uint64              `json:"curtidas"`
	Comentarios         uint64              `json:"comentarios"`
	CurtidoPorUsuario   bool                `json:"curtidoPorUsuario"`
	SalvoPorUsuario     bool                `json:"salvoPorUsuario"`
	CriadaEm            time.Time           `json:"criadaEm,omitempty"`
	Editado             bool                `json:"editado"`
	TemImagem           bool                `json:"temImagem"`
	Reposts             uint64              `json:"reposts"`
	RepostadoPorUsuario bool                `json:"repostadoPorUsuario"`
	RepostDe            *PublicacaoOriginal `json:"repostDe,omitempty"`
}

// PublicacaoOriginal é a publicação repostada, embutida no repost. Indisponivel vale quando quem vê não tem acesso a ela.
type PublicacaoOriginal struct {
	ID              uint64    `json:"id"`
	AutorID         uint64    `json:"autorId,omitempty"`
	AutorNick       string    `json:"autorNick,omitempty"`
	AutorFotoVersao uint64    `json:"autorFotoVersao,omitempty"`
	Titulo          string    `json:"titulo,omitempty"`
	Conteudo        string    `json:"conteudo,omitempty"`
	CriadaEm        time.Time `json:"criadaEm,omitempty"`
	TemImagem       bool      `json:"temImagem,omitempty"`
	Indisponivel    bool      `json:"indisponivel,omitempty"`
}

// Preparar vai chamar os metodos para validar e formatar a publicacao do usuario
func (publicacao *Publicacao) Preparar() error {
	if erro := publicacao.validar(); erro != nil {
		return erro
	}

	publicacao.formatar()
	return nil
}

func (publicacao *Publicacao) validar() error {
	if publicacao.Conteudo == "" {
		return errors.New("O Conteudo é obrigatorio e não pode estar em branco")
	}

	if utf8.RuneCountInString(publicacao.Titulo) > 150 {
		return errors.New("O Titulo deve ter no máximo 150 caracteres")
	}

	if utf8.RuneCountInString(publicacao.Conteudo) > 5000 {
		return errors.New("O Conteudo deve ter no máximo 5000 caracteres")
	}
	return nil
}

func (publicacao *Publicacao) formatar() {
	publicacao.Titulo = strings.TrimSpace(publicacao.Titulo)
	publicacao.Conteudo = strings.TrimSpace(publicacao.Conteudo)
}
