package repositorios

import (
	"orby/api/src/eventos"
	"orby/api/src/testes"
	"testing"
	"time"
)

func TestAvisarNovaPublicacaoChegaSoAosSeguidoresQueNaoSilenciaram(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	seguidor := testes.CriarUsuario(t, db, "seguidor", false)
	silenciou := testes.CriarUsuario(t, db, "silenciou", false)
	estranho := testes.CriarUsuario(t, db, "estranho", false)
	db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?), (?, ?)", autor, seguidor, autor, silenciou)
	db.Exec("insert into silenciados (silenciador_id, silenciado_id) values (?, ?)", silenciou, autor)

	doSeguidor, sair1 := eventos.Assinar(seguidor)
	defer sair1()
	doSilenciou, sair2 := eventos.Assinar(silenciou)
	defer sair2()
	doEstranho, sair3 := eventos.Assinar(estranho)
	defer sair3()

	if erro := NovoRepositorioDePublicacoes(db).AvisarNovaPublicacao(autor); erro != nil {
		t.Fatal(erro)
	}

	select {
	case e := <-doSeguidor:
		if e.Tipo != "publicacao" || e.De != autor {
			t.Fatalf("evento inesperado: %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("o seguidor deveria ser avisado")
	}
	select {
	case e := <-doSilenciou:
		t.Fatalf("quem silenciou não deveria ser avisado: %+v", e)
	case e := <-doEstranho:
		t.Fatalf("quem não segue não deveria ser avisado: %+v", e)
	case <-time.After(100 * time.Millisecond):
	}
}
