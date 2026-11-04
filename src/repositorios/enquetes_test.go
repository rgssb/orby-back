package repositorios

import (
	"errors"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"strings"
	"sync"
	"testing"
)

func TestVotosConcorrentesETransacaoDaEnquete(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	votante := testes.CriarUsuario(t, db, "votante", false)
	repo := NovoRepositorioDePublicacoes(db)
	_, erro := repo.Criar(modelos.Publicacao{
		AutorID: autor, Conteudo: "inválida",
		Enquete: &modelos.Enquete{Opcoes: []modelos.OpcaoEnquete{{Texto: strings.Repeat("x", 141)}}},
	})
	if erro == nil {
		t.Fatal("falha ao gravar opção deveria cancelar a publicação")
	}
	var quantidade int
	if erro = db.QueryRow("select count(*) from publicacoes").Scan(&quantidade); erro != nil || quantidade != 0 {
		t.Fatalf("publicação não foi revertida: %d %v", quantidade, erro)
	}
	id, erro := repo.Criar(modelos.Publicacao{
		AutorID: autor, Conteudo: "pergunta",
		Enquete: &modelos.Enquete{Opcoes: []modelos.OpcaoEnquete{{Texto: "a"}, {Texto: "b"}}},
	})
	if erro != nil {
		t.Fatal(erro)
	}
	linhas, erro := db.Query("select id from opcoes_enquetes where publicacao_id = ? order by id", id)
	if erro != nil {
		t.Fatal(erro)
	}
	var opcoes []uint64
	for linhas.Next() {
		var opcao uint64
		if erro = linhas.Scan(&opcao); erro != nil {
			t.Fatal(erro)
		}
		opcoes = append(opcoes, opcao)
	}
	if erro = linhas.Err(); erro != nil {
		t.Fatal(erro)
	}
	linhas.Close()
	if len(opcoes) != 2 {
		t.Fatal("opções ausentes")
	}
	resultados := make(chan error, 2)
	var grupo sync.WaitGroup
	for _, opcao := range opcoes {
		grupo.Add(1)
		go func(opcao uint64) {
			defer grupo.Done()
			resultados <- repo.VotarEnquete(votante, id, opcao)
		}(opcao)
	}
	grupo.Wait()
	close(resultados)
	sucesso, duplicado := 0, 0
	for erro := range resultados {
		switch {
		case erro == nil:
			sucesso++
		case errors.Is(erro, ErrVotoExistente):
			duplicado++
		default:
			t.Fatal(erro)
		}
	}
	if sucesso != 1 || duplicado != 1 {
		t.Fatalf("votos concorrentes: %d aceitos, %d duplicados", sucesso, duplicado)
	}
}
