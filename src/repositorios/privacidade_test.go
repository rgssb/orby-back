package repositorios

import (
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
)

func idsDasPublicacoes(publicacoes []modelos.Publicacao) map[uint64]bool {
	ids := map[uint64]bool{}
	for _, p := range publicacoes {
		ids[p.ID] = true
	}
	return ids
}

func TestPerfilPrivadoEscondePublicacoes(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	seguidor := testes.CriarUsuario(t, db, "seguidor", false)
	estranho := testes.CriarUsuario(t, db, "estranho", false)

	publicacoes := NovoRepositorioDePublicacoes(db)
	postID, erro := publicacoes.Criar(modelos.Publicacao{Titulo: "segredo", Conteudo: "so para quem segue", AutorID: dono})
	if erro != nil {
		t.Fatal(erro)
	}
	if erro = NovoRepositorioDeUsuarios(db).Seguir(dono, seguidor); erro != nil {
		t.Fatal(erro)
	}

	casos := []struct {
		nome   string
		viewer uint64
		ve     bool
	}{
		{"dono", dono, true},
		{"seguidor", seguidor, true},
		{"estranho", estranho, false},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			feed, erro := publicacoes.Buscar(c.viewer, 0, 0, "todos", "recentes")
			if erro != nil {
				t.Fatal(erro)
			}
			if idsDasPublicacoes(feed)[postID] != c.ve {
				t.Errorf("feed 'todos': visível = %v, esperado %v", !c.ve, c.ve)
			}

			busca, erro := publicacoes.Pesquisar(c.viewer, "segredo", false, "", 10, 0)
			if erro != nil {
				t.Fatal(erro)
			}
			if idsDasPublicacoes(busca)[postID] != c.ve {
				t.Errorf("pesquisa: visível = %v, esperado %v", !c.ve, c.ve)
			}
		})
	}
}

func TestPedidoParaSeguirPerfilPrivado(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	visitante := testes.CriarUsuario(t, db, "visitante", false)
	usuarios := NovoRepositorioDeUsuarios(db)

	if erro := usuarios.PedirParaSeguir(dono, visitante); erro != nil {
		t.Fatal(erro)
	}
	privado, segue, pedido, erro := usuarios.Privacidade(dono, visitante)
	if erro != nil || !privado || segue || !pedido {
		t.Fatalf("após pedir: privado=%v segue=%v pedido=%v erro=%v", privado, segue, pedido, erro)
	}

	aceito, erro := usuarios.AceitarPedido(dono, visitante)
	if erro != nil || !aceito {
		t.Fatalf("aceitar deveria funcionar: aceito=%v erro=%v", aceito, erro)
	}
	_, segue, pedido, _ = usuarios.Privacidade(dono, visitante)
	if !segue || pedido {
		t.Fatalf("após aceitar: segue=%v pedido=%v", segue, pedido)
	}

	if aceito, _ = usuarios.AceitarPedido(dono, visitante); aceito {
		t.Fatal("aceitar sem pedido pendente deveria devolver false")
	}
}

func TestPedidoCancelado(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	visitante := testes.CriarUsuario(t, db, "visitante", false)
	usuarios := NovoRepositorioDeUsuarios(db)

	usuarios.PedirParaSeguir(dono, visitante)
	if erro := usuarios.PararDeSeguir(dono, visitante); erro != nil {
		t.Fatal(erro)
	}
	if aceito, _ := usuarios.AceitarPedido(dono, visitante); aceito {
		t.Fatal("pedido cancelado não pode ser aceito")
	}
}

func TestAbrirPerfilAceitaPedidosPendentes(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	visitante := testes.CriarUsuario(t, db, "visitante", false)
	usuarios := NovoRepositorioDeUsuarios(db)

	usuarios.PedirParaSeguir(dono, visitante)
	if erro := usuarios.DefinirPrivacidade(dono, false); erro != nil {
		t.Fatal(erro)
	}
	privado, segue, pedido, _ := usuarios.Privacidade(dono, visitante)
	if privado || !segue || pedido {
		t.Fatalf("privado=%v segue=%v pedido=%v", privado, segue, pedido)
	}
}
