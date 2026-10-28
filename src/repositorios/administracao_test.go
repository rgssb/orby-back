package repositorios

import (
	"fmt"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
)

func TestIdentificacaoDeAdmin(t *testing.T) {
	db := testes.NovoBanco(t)
	admin := testes.CriarUsuario(t, db, "admin", false)
	pessoa := testes.CriarUsuario(t, db, "pessoa", false)
	if _, erro := db.Exec("update usuarios set admin = 1 where id = ?", admin); erro != nil {
		t.Fatal(erro)
	}
	perfil, erro := NovoRepositorioDeUsuarios(db).BuscarPorID(admin)
	if erro != nil || !perfil.Admin {
		t.Fatalf("perfil: %+v, %v", perfil, erro)
	}
	resultado, erro := db.Exec("insert into publicacoes (autor_id, conteudo) values (?, 'teste')", pessoa)
	if erro != nil {
		t.Fatal(erro)
	}
	publicacaoID, _ := resultado.LastInsertId()
	comentarios := NovoRepositorioDeComentarios(db)
	if _, erro = comentarios.Criar(modelos.Comentario{PublicacaoID: uint64(publicacaoID), AutorID: admin, Conteudo: "teste"}); erro != nil {
		t.Fatal(erro)
	}
	lista, erro := comentarios.BuscarPorPublicacao(uint64(publicacaoID), pessoa, 10, 0)
	if erro != nil || len(lista) != 1 || !lista[0].AutorAdmin {
		t.Fatalf("comentários: %+v, %v", lista, erro)
	}
	dialogo, erro := NovoRepositorioDeMensagens(db).Abrir(pessoa, admin, 0, 0)
	if erro != nil || !dialogo.Admin {
		t.Fatalf("diálogo: %+v, %v", dialogo, erro)
	}
	grupos := NovoRepositorioDeGrupos(db)
	grupoID, erro := grupos.Criar(admin, "Teste", []uint64{pessoa})
	if erro != nil {
		t.Fatal(erro)
	}
	mensagem, erro := grupos.Enviar(admin, grupoID, "teste")
	if erro != nil || !mensagem.Admin {
		t.Fatalf("mensagem: %+v, %v", mensagem, erro)
	}
	grupo, erro := grupos.Abrir(pessoa, grupoID, 0)
	if erro != nil || len(grupo.MensagensDoGrupo) != 1 || !grupo.MensagensDoGrupo[0].Admin {
		t.Fatalf("grupo: %+v, %v", grupo, erro)
	}
}

func TestListagensAdministrativas(t *testing.T) {
	db := testes.NovoBanco(t)
	repo := NovoRepositorioDeAdministracao(db)
	for i := 0; i < 17; i++ {
		testes.CriarUsuario(t, db, fmt.Sprintf("pessoa%02d", i), false)
	}
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	denunciante := testes.CriarUsuario(t, db, "denunciante", false)
	if _, erro := db.Exec("update usuarios set criadoEm = now() - interval 8 day, suspenso_em = now() where id = ?", alvo); erro != nil {
		t.Fatal(erro)
	}
	if erro := NovoRepositorioDeDenuncias(db).Registrar(denunciante, modelos.DenunciaPerfil, alvo, "spam"); erro != nil {
		t.Fatal(erro)
	}
	publicacao, erro := db.Exec("insert into publicacoes (titulo, conteudo, autor_id) values ('Teste', 'Texto', ?)", alvo)
	if erro != nil {
		t.Fatal(erro)
	}
	id, _ := publicacao.LastInsertId()
	if _, erro = db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'Resposta')", id, alvo); erro != nil {
		t.Fatal(erro)
	}
	primeira, erro := repo.Listar("contas", "pessoa", 0)
	if erro != nil || len(primeira) != 16 {
		t.Fatalf("primeira página: %d, %v", len(primeira), erro)
	}
	segunda, erro := repo.Listar("contas", "pessoa", 15)
	if erro != nil || len(segunda) != 2 || segunda[0].ID != primeira[15].ID {
		t.Fatalf("segunda página: %+v, %v", segunda, erro)
	}
	for _, caso := range []struct {
		tipo, busca string
		total       int
	}{
		{"novas", "alvo", 0},
		{"suspensas", "alvo", 1},
		{"denunciadas", "alvo", 1},
		{"publicacoes", "Texto", 1},
		{"comentarios", "Resposta", 1},
	} {
		itens, erro := repo.Listar(caso.tipo, caso.busca, 0)
		if erro != nil || len(itens) != caso.total {
			t.Errorf("%s: %+v, %v", caso.tipo, itens, erro)
		}
		if len(itens) > 0 && !itens[0].Suspenso {
			t.Errorf("%s: suspensão não informada", caso.tipo)
		}
	}
}
