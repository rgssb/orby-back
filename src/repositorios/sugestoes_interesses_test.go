package repositorios

import (
	"orby/api/src/testes"
	"testing"
)

func TestSugestoesPorHashtagNaoUsamConteudoRestrito(t *testing.T) {
	db := testes.NovoBanco(t)
	eu := testes.CriarUsuario(t, db, "eu", false)
	fonte := testes.CriarUsuario(t, db, "fonte", false)
	mesmoInteresse := testes.CriarUsuario(t, db, "mesmointeresse", false)
	exclusivo := testes.CriarUsuario(t, db, "exclusivo", false)
	privado := testes.CriarUsuario(t, db, "privado", true)
	if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", fonte, eu); erro != nil {
		t.Fatal(erro)
	}
	for _, pessoa := range []uint64{fonte, mesmoInteresse, exclusivo, privado} {
		resultado, erro := db.Exec("insert into publicacoes (autor_id, conteudo) values (?, 'assunto #Arte')", pessoa)
		if erro != nil {
			t.Fatal(erro)
		}
		if pessoa == exclusivo {
			id, _ := resultado.LastInsertId()
			if _, erro = db.Exec("insert into publicacoes_felas (publicacao_id) values (?)", id); erro != nil {
				t.Fatal(erro)
			}
		}
	}
	sugestoes, erro := NovoRepositorioDeUsuarios(db).SugerirSeguir(eu, 5)
	if erro != nil {
		t.Fatal(erro)
	}
	if len(sugestoes) != 3 || sugestoes[0].ID != mesmoInteresse || sugestoes[0].Motivo != "interesses" {
		t.Fatalf("interesse não priorizado: %+v", sugestoes)
	}
	for _, pessoa := range sugestoes {
		if (pessoa.ID == exclusivo || pessoa.ID == privado) && pessoa.Motivo == "interesses" {
			t.Fatalf("conteúdo restrito influenciou sugestões: %+v", sugestoes)
		}
	}
}
