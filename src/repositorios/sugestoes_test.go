package repositorios

import (
	"orby/api/src/testes"
	"testing"
)

func TestSugerirSeguirPriorizaAmigosEmComumEExcluiBloqueadosEQuemJaSegue(t *testing.T) {
	db := testes.NovoBanco(t)
	eu := testes.CriarUsuario(t, db, "eu", false)
	amigo1 := testes.CriarUsuario(t, db, "amigo1", false)
	amigo2 := testes.CriarUsuario(t, db, "amigo2", false)
	popular := testes.CriarUsuario(t, db, "popular", false)
	poucoComum := testes.CriarUsuario(t, db, "poucocomum", false)
	jaSigo := testes.CriarUsuario(t, db, "jasigo", false)
	bloqueado := testes.CriarUsuario(t, db, "bloqueado", false)
	semEmail := testes.CriarUsuario(t, db, "sememail", false)
	db.Exec("update usuarios set email_verificado_em = null where id = ?", semEmail)

	seguir := func(quem, alvo uint64) {
		if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", alvo, quem); erro != nil {
			t.Fatal(erro)
		}
	}
	seguir(eu, amigo1)
	seguir(eu, amigo2)
	seguir(eu, jaSigo)
	seguir(amigo1, popular)
	seguir(amigo2, popular)
	seguir(amigo1, poucoComum)
	seguir(amigo1, jaSigo)
	seguir(amigo1, bloqueado)
	seguir(amigo1, semEmail)
	db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", bloqueado, eu)

	sugestoes, erro := NovoRepositorioDeUsuarios(db).SugerirSeguir(eu, 5)
	if erro != nil {
		t.Fatal(erro)
	}

	if len(sugestoes) != 2 || sugestoes[0].ID != popular || sugestoes[1].ID != poucoComum {
		t.Fatalf("sugestões inesperadas: %+v", sugestoes)
	}
	if sugestoes[0].EmComum != 2 || sugestoes[1].EmComum != 1 || sugestoes[1].Via != "amigo1" {
		t.Fatalf("contagem em comum errada: %+v", sugestoes)
	}
}

func TestSugerirSeguirCompletaComMaisSeguidosSemRepetir(t *testing.T) {
	db := testes.NovoBanco(t)
	eu := testes.CriarUsuario(t, db, "eu", false)
	amigo := testes.CriarUsuario(t, db, "amigo", false)
	comum := testes.CriarUsuario(t, db, "comum", false)
	famoso := testes.CriarUsuario(t, db, "famoso", false)
	fa := testes.CriarUsuario(t, db, "fa", false)
	db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?), (?, ?), (?, ?)", amigo, eu, comum, amigo, famoso, fa)

	sugestoes, erro := NovoRepositorioDeUsuarios(db).SugerirSeguir(eu, 3)
	if erro != nil {
		t.Fatal(erro)
	}

	if len(sugestoes) != 3 || sugestoes[0].ID != comum || sugestoes[1].ID != famoso {
		t.Fatalf("sugestões inesperadas: %+v", sugestoes)
	}
	if sugestoes[1].EmComum != 0 {
		t.Fatalf("preenchimento não deveria ter pessoas em comum: %+v", sugestoes[1])
	}
}
