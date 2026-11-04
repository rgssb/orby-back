package repositorios

import (
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
)

func TestDescricoesDasImagensSobrevivemAoRepostEAPromocao(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	repo := NovoRepositorioDePublicacoes(db)
	id, erro := repo.Criar(modelos.Publicacao{Conteudo: "Olá", AutorID: autor})
	if erro != nil {
		t.Fatal(erro)
	}
	for _, descricao := range []string{"Um gato", "Um cão"} {
		if erro = repo.AdicionarImagemComDescricao(id, "image/png", []byte("imagem"), descricao); erro != nil {
			t.Fatal(erro)
		}
	}
	repost, erro := repo.Repostar(autor, id, "")
	if erro != nil {
		t.Fatal(erro)
	}
	consultar := func() []modelos.Publicacao {
		publicacoes := []modelos.Publicacao{{ID: id}, {ID: repost}}
		if erro := repo.AnexarRepostagens(publicacoes, autor); erro != nil {
			t.Fatal(erro)
		}
		return publicacoes
	}
	publicacoes := consultar()
	if len(publicacoes[0].DescricoesImagens) != 2 || publicacoes[0].DescricoesImagens[0] != "Um gato" ||
		publicacoes[1].RepostDe == nil || len(publicacoes[1].RepostDe.DescricoesImagens) != 2 ||
		publicacoes[1].RepostDe.DescricoesImagens[1] != "Um cão" {
		t.Fatalf("descrições incorretas: %+v", publicacoes)
	}
	if erro = repo.RemoverImagem(id, 0); erro != nil {
		t.Fatal(erro)
	}
	publicacoes = consultar()
	if len(publicacoes[0].DescricoesImagens) != 1 || publicacoes[0].DescricoesImagens[0] != "Um cão" ||
		len(publicacoes[1].RepostDe.DescricoesImagens) != 1 || publicacoes[1].RepostDe.DescricoesImagens[0] != "Um cão" {
		t.Fatalf("descrição não acompanhou a imagem promovida: %+v", publicacoes)
	}
}
