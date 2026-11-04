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
	AutorSuspenso       bool                `json:"autorSuspenso"`
	AutorFotoVersao     uint64              `json:"autorFotoVersao"`
	Curtidas            uint64              `json:"curtidas"`
	Comentarios         uint64              `json:"comentarios"`
	CurtidoPorUsuario   bool                `json:"curtidoPorUsuario"`
	SalvoPorUsuario     bool                `json:"salvoPorUsuario"`
	CriadaEm            time.Time           `json:"criadaEm,omitempty"`
	Editado             bool                `json:"editado"`
	TemImagem           bool                `json:"temImagem"`
	Imagens             int                 `json:"imagens"`
	DescricoesImagens   []string            `json:"descricoesImagens,omitempty"`
	Reposts             uint64              `json:"reposts"`
	RepostadoPorUsuario bool                `json:"repostadoPorUsuario"`
	RepostDe            *PublicacaoOriginal `json:"repostDe,omitempty"`
	AcessoModeracao     bool                `json:"acessoModeracao"`
	Felas               bool                `json:"felas"`
	Enquete             *Enquete            `json:"enquete,omitempty"`
}

type Enquete struct {
	Titulo     string         `json:"titulo,omitempty"`
	Opcoes     []OpcaoEnquete `json:"opcoes"`
	EncerraEm  *time.Time     `json:"encerraEm,omitempty"`
	Encerrada  bool           `json:"encerrada"`
	Votou      bool           `json:"votou"`
	TotalVotos uint64         `json:"totalVotos"`
}

type OpcaoEnquete struct {
	ID    uint64 `json:"id,omitempty"`
	Texto string `json:"texto"`
	Votos uint64 `json:"votos"`
	Minha bool   `json:"minha"`
}

// PublicacaoOriginal é a publicação repostada, embutida no repost. Indisponivel vale quando quem vê não tem acesso a ela.
type PublicacaoOriginal struct {
	ID                uint64    `json:"id"`
	AutorID           uint64    `json:"autorId,omitempty"`
	AutorNick         string    `json:"autorNick,omitempty"`
	AutorFotoVersao   uint64    `json:"autorFotoVersao,omitempty"`
	Titulo            string    `json:"titulo,omitempty"`
	Conteudo          string    `json:"conteudo,omitempty"`
	CriadaEm          time.Time `json:"criadaEm,omitempty"`
	Imagens           int       `json:"imagens,omitempty"`
	DescricoesImagens []string  `json:"descricoesImagens,omitempty"`
	Indisponivel      bool      `json:"indisponivel,omitempty"`
}

// Preparar vai chamar os metodos para validar e formatar a publicacao do usuario
func (publicacao *Publicacao) Preparar() error {
	if erro := publicacao.validar(); erro != nil {
		return erro
	}

	publicacao.formatar()
	if publicacao.Enquete != nil {
		e := publicacao.Enquete
		e.Titulo = strings.TrimSpace(e.Titulo)
		if utf8.RuneCountInString(e.Titulo) > 150 {
			return errors.New("O título da enquete deve ter no máximo 150 caracteres")
		}
		if len(e.Opcoes) < 2 || len(e.Opcoes) > 10 {
			return errors.New("A enquete deve ter entre 2 e 10 opções")
		}
		if e.EncerraEm != nil && !e.EncerraEm.After(time.Now()) {
			return errors.New("O encerramento da enquete deve estar no futuro")
		}
		nomes := make(map[string]bool)
		for i := range e.Opcoes {
			texto := strings.TrimSpace(e.Opcoes[i].Texto)
			if texto == "" || utf8.RuneCountInString(texto) > 140 || nomes[strings.ToLower(texto)] {
				return errors.New("As opções devem ser diferentes e ter de 1 a 140 caracteres")
			}
			nomes[strings.ToLower(texto)] = true
			e.Opcoes[i] = OpcaoEnquete{Texto: texto}
		}
	}
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
