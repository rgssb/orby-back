package modelos

import (
	"errors"
	"orby/api/src/seguranca"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/badoux/checkmail"
)

// Usuario representa um usuario ativo da rede social
type Usuario struct {
	ID    uint64 `json:"id,omitempty"`
	Nome  string `json:"nome,omitempty"`
	Nick  string `json:"nick,omitempty"`
	Email string `json:"email,omitempty"`
	Senha string `json:"senha,omitempty"`
	// AceitouTermos é exigido só no cadastro
	AceitouTermos bool   `json:"aceitouTermos,omitempty"`
	Bio           string `json:"bio"`
	FotoVersao    uint64 `json:"fotoVersao"`
	// BannerPreset é o tema pronto escolhido para o banner ("" quando não há) e BannerVersao é diferente de 0 quando há imagem própria
	BannerPreset      string    `json:"bannerPreset"`
	BannerVersao      uint64    `json:"bannerVersao"`
	CriadoEm          time.Time `json:"criadoEm,omitempty"`
	Admin             bool      `json:"admin"`
	Suspenso          bool      `json:"-"`
	SuspensoParaAdmin *bool     `json:"suspenso,omitempty"`
	// Privado esconde as publicações de quem não segue; os dois campos seguintes dependem de quem visita o perfil
	Privado            bool `json:"privado"`
	PodeVerPublicacoes bool `json:"podeVerPublicacoes"`
	PedidoEnviado      bool `json:"pedidoEnviado"`
	AcessoModeracao    bool `json:"acessoModeracao"`
}

// TamanhoMinimoSenha é a quantidade mínima de caracteres de uma senha
const TamanhoMinimoSenha = 6

// ValidarNovaSenha confere se a senha atende aos requisitos mínimos
func ValidarNovaSenha(senha string) error {
	tamanho := utf8.RuneCountInString(senha)
	if tamanho < TamanhoMinimoSenha {
		return errors.New("A senha deve ter pelo menos 6 caracteres")
	}
	if len(senha) > 72 {
		return errors.New("A senha deve ter no máximo 72 caracteres")
	}
	return nil
}

// Preparar vai chamar os metodos para validar e formatar o usuario recebido
func (usuario *Usuario) Preparar(etapa string) error {
	if erro := usuario.validar(etapa); erro != nil {
		return erro
	}

	if erro := usuario.formatar(etapa); erro != nil {
		return erro
	}

	return nil
}

func (usuario *Usuario) validar(etapa string) error {
	if usuario.Nome == "" {
		return errors.New("O nome é obrigatorio, não será possivel deixa-lo em branco")
	}

	if usuario.Nick == "" {
		return errors.New("O nick é obrigatorio, não será possivel deixa-lo em branco")
	}

	if usuario.Email == "" {
		return errors.New("O email é obrigatorio, não será possivel deixa-lo em branco")
	}

	if erro := checkmail.ValidateFormat(usuario.Email); erro != nil {
		return errors.New("O email inserido é invalido")
	}

	if etapa == "cadastro" && usuario.Senha == "" {
		return errors.New("A senha é obrigatoria, não será possivel deixa-la em branco")
	}

	if etapa == "cadastro" {
		if erro := ValidarNovaSenha(usuario.Senha); erro != nil {
			return erro
		}
	}

	if utf8.RuneCountInString(usuario.Nome) > 50 {
		return errors.New("O nome deve ter no máximo 50 caracteres")
	}

	if utf8.RuneCountInString(usuario.Nick) > 50 {
		return errors.New("O nick deve ter no máximo 50 caracteres")
	}

	if utf8.RuneCountInString(usuario.Email) > 50 {
		return errors.New("O email deve ter no máximo 50 caracteres")
	}

	if utf8.RuneCountInString(strings.TrimSpace(usuario.Bio)) > 160 {
		return errors.New("A bio deve ter no máximo 160 caracteres")
	}

	return nil

}

func (usuario *Usuario) formatar(etapa string) error {
	usuario.Nome = strings.TrimSpace(usuario.Nome)
	usuario.Nick = strings.TrimSpace(usuario.Nick)
	usuario.Email = strings.TrimSpace(usuario.Email)
	usuario.Bio = strings.TrimSpace(usuario.Bio)

	if etapa == "cadastro" {
		senhaComHash, erro := seguranca.Hash(usuario.Senha)
		if erro != nil {
			return erro
		}

		usuario.Senha = string(senhaComHash)

	}
	return nil
}
