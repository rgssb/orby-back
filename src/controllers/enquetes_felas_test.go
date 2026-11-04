package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
)

func TestEnqueteFelasExigeAcessoAoVotar(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	membro := testes.CriarUsuario(t, db, "membro", false)
	if _, erro := db.Exec("insert into felas (dono_id, membro_id) values (?, ?)", autor, membro); erro != nil {
		t.Fatal(erro)
	}
	criar := chamar(t, autor, http.MethodPost, "/publicacoes", `{"conteudo":"só para Felas","felas":true,"enquete":{"opcoes":[{"texto":"a"},{"texto":"b"}]}}`)
	var post modelos.Publicacao
	if criar.Code != http.StatusCreated || json.Unmarshal(criar.Body.Bytes(), &post) != nil {
		t.Fatalf("criação: %d %s", criar.Code, criar.Body)
	}
	var opcao uint64
	if erro := db.QueryRow("select id from opcoes_enquetes where publicacao_id = ? limit 1", post.ID).Scan(&opcao); erro != nil {
		t.Fatal(erro)
	}
	caminho := fmt.Sprintf("/publicacoes/%d/enquete/%d/votar", post.ID, opcao)
	if _, erro := db.Exec("delete from felas where dono_id = ? and membro_id = ?", autor, membro); erro != nil {
		t.Fatal(erro)
	}
	if resposta := chamar(t, membro, http.MethodPost, caminho, ""); resposta.Code != http.StatusNotFound {
		t.Fatalf("voto após remoção de Felas: %d", resposta.Code)
	}
}
