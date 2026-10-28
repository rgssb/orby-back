package repositorios

import (
	"orby/api/src/modelos"
	"orby/api/src/suspensoes"
	"orby/api/src/testes"
	"testing"
)

func TestReativarFalhaSemArquivarOuLiberarConta(t *testing.T) {
	db := testes.NovoBanco(t)
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	t.Cleanup(func() { suspensoes.Definir(alvo, false) })
	pessoa := testes.CriarUsuario(t, db, "pessoa", false)
	denuncias := NovoRepositorioDeDenuncias(db)
	admin := NovoRepositorioDeAdministracao(db)
	if erro := denuncias.Registrar(pessoa, modelos.DenunciaPerfil, alvo, "spam"); erro != nil {
		t.Fatal(erro)
	}
	if erro := admin.Suspender(alvo); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := db.Exec("drop table notificacoes"); erro != nil {
		t.Fatal(erro)
	}
	if erro := admin.Reativar(alvo); erro == nil {
		t.Fatal("falha ao arquivar deve impedir a reativação")
	}
	var suspensa, aberta bool
	if erro := db.QueryRow("select suspenso_em is not null from usuarios where id = ?", alvo).Scan(&suspensa); erro != nil {
		t.Fatal(erro)
	}
	if erro := db.QueryRow("select resolvida = 0 from denuncias where denunciante_id = ? and alvo_id = ?", pessoa, alvo).Scan(&aberta); erro != nil {
		t.Fatal(erro)
	}
	if !suspensa || !aberta {
		t.Fatalf("reativação parcial: suspensa=%v aberta=%v", suspensa, aberta)
	}
}
