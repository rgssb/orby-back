package repositorios

import (
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
)

func TestPesquisarComFiltros(t *testing.T) {
	db := testes.NovoBanco(t)
	visitante := testes.CriarUsuario(t, db, "visitante", false)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	privado := testes.CriarUsuario(t, db, "privado", true)
	repo := NovoRepositorioDePublicacoes(db)

	criar := func(autor uint64, conteudo, data string, felas bool) uint64 {
		t.Helper()
		id, err := repo.Criar(modelos.Publicacao{AutorID: autor, Conteudo: conteudo, Felas: felas})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec("update publicacoes set criadoEm = ? where id = ?", data, id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	primeiro := criar(ana, "olá #golang", "2024-01-02 00:00:00", false)
	ultimo := criar(ana, "olá #golang", "2024-01-03 23:59:59", false)
	criar(ana, "olá #golang", "2024-01-04 00:00:00", false)
	criar(beto, "olá #golang", "2024-01-03 12:00:00", false)
	criar(privado, "olá #golang", "2024-01-03 12:00:00", false)
	criar(ana, "olá #golang", "2024-01-03 12:00:00", true)
	criar(ana, "olá #golangextra", "2024-01-03 12:00:00", false)

	resultados, err := repo.PesquisarComFiltros(visitante, "golang", true, "", "ana", "2024-01-02", "2024-01-03", 10, 0)
	if err != nil || len(resultados) != 2 || resultados[0].ID != ultimo || resultados[1].ID != primeiro {
		t.Fatalf("filtros de hashtag, autor e datas: %+v, %v", resultados, err)
	}
	resultados, err = repo.PesquisarComFiltros(visitante, "", false, "", "ana", "2024-01-02", "2024-01-03", 1, 1)
	if err != nil || len(resultados) != 1 || resultados[0].ID != ultimo {
		t.Fatalf("filtro sem termo e com paginação: %+v, %v", resultados, err)
	}
	resultados, err = repo.PesquisarComFiltros(visitante, "", false, "", "privado", "", "", 10, 0)
	if err != nil || len(resultados) != 0 {
		t.Fatalf("perfil privado ficou visível: %+v, %v", resultados, err)
	}
}
