package repositorios

import (
	"orby/api/src/testes"
	"testing"
)

func TestEmAltaOrdenaPorComentariosEIgnoraPostsAntigos(t *testing.T) {
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

	muitasCurtidas := novo(50, false)
	maisComentado := novo(0, false)
	comentar(maisComentado, 3)
	antigo := novo(0, true)
	comentar(antigo, 10)

	publicacoes, erro := NovoRepositorioDePublicacoes(db).Buscar(autor, 0, 0, "todos", "populares")
	if erro != nil {
		t.Fatal(erro)
	}
	if len(publicacoes) != 2 || publicacoes[0].ID != maisComentado || publicacoes[1].ID != muitasCurtidas {
		t.Fatalf("ordem inesperada (antigo=%d): %+v", antigo, idsDasPublicacoes(publicacoes))
	}
}
