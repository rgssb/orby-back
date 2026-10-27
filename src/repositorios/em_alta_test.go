package repositorios

import (
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
)

func TestEmAltaSomaCurtidasEComentariosEIgnoraPostsAntigos(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)

	novo := func(curtidas int, antigo bool) uint64 {
		criado := "now()"
		if antigo {
			criado = "now() - interval 8 day"
		}
		r, erro := db.Exec("insert into publicacoes (titulo, conteudo, autor_id, curtidas, criadoEm) values ('', 'x', ?, ?, "+criado+")", autor, curtidas)
		if erro != nil {
			t.Fatal(erro)
		}
		id, _ := r.LastInsertId()
		return uint64(id)
	}
	comentar := func(post uint64, vezes int) {
		for i := 0; i < vezes; i++ {
			db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'oi')", post, autor)
		}
	}

	maisCurtido := novo(5, false)
	maisComentado := novo(0, false)
	comentar(maisComentado, 6)
	misto := novo(2, false)
	comentar(misto, 2)
	antigo := novo(0, true)
	comentar(antigo, 10)

	publicacoes, erro := NovoRepositorioDePublicacoes(db).Buscar(autor, 0, 0, "todos", "populares")
	if erro != nil {
		t.Fatal(erro)
	}
	// pontuação: comentado 6, curtido 5, misto 4
	if len(publicacoes) != 3 || publicacoes[0].ID != maisComentado || publicacoes[1].ID != maisCurtido || publicacoes[2].ID != misto {
		t.Fatalf("ordem inesperada (antigo=%d): %+v", antigo, idsDasPublicacoes(publicacoes))
	}
}

func TestHashtagEmAltaOrdenaPorCurtidasEComentarios(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	publicacoes := NovoRepositorioDePublicacoes(db)
	antiga, _ := publicacoes.Criar(modelos.Publicacao{Conteudo: "primeira #go", AutorID: autor})
	nova, _ := publicacoes.Criar(modelos.Publicacao{Conteudo: "segunda #go", AutorID: autor})
	publicacoes.Criar(modelos.Publicacao{Conteudo: "outra #gopher", AutorID: autor})
	db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'oi')", antiga, autor)

	recentes, erro := publicacoes.Pesquisar(autor, "go", true, "", 10, 0)
	if erro != nil || len(recentes) != 2 || recentes[0].ID != nova {
		t.Fatalf("recentes: %+v %v", recentes, erro)
	}
	populares, erro := publicacoes.Pesquisar(autor, "go", true, "populares", 10, 0)
	if erro != nil || len(populares) != 2 || populares[0].ID != antiga {
		t.Fatalf("populares: %+v %v", populares, erro)
	}
}
