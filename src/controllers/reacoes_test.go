package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
)

func TestReacoesPublicacoesEComentarios(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", true)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	for _, usuario := range []uint64{ana, beto} {
		if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", autor, usuario); erro != nil {
			t.Fatal(erro)
		}
	}
	post := chamar(t, autor, http.MethodPost, "/publicacoes", `{"conteudo":"olá"}`)
	var publicacao modelos.Publicacao
	if post.Code != http.StatusCreated || json.Unmarshal(post.Body.Bytes(), &publicacao) != nil {
		t.Fatalf("publicação: %d %s", post.Code, post.Body)
	}
	resultado, erro := db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'comentário')", publicacao.ID, autor)
	if erro != nil {
		t.Fatal(erro)
	}
	comentario, _ := resultado.LastInsertId()
	for _, caminho := range []string{
		fmt.Sprintf("/publicacoes/%d/reacoes", publicacao.ID),
		fmt.Sprintf("/comentarios/%d/reacoes", comentario),
	} {
		for _, usuario := range []uint64{ana, beto} {
			resposta := chamar(t, usuario, http.MethodPut, caminho, `{"emoji":"😂","ativa":true}`)
			if resposta.Code != http.StatusNoContent {
				t.Fatalf("adicionar: %d %s", resposta.Code, resposta.Body)
			}
		}
		if resposta := chamar(t, ana, http.MethodPut, caminho, `{"emoji":"😂","ativa":true}`); resposta.Code != http.StatusNoContent {
			t.Fatalf("repetir: %d %s", resposta.Code, resposta.Body)
		}
		resposta := chamar(t, ana, http.MethodGet, caminho, "")
		var reacoes []modelos.Reacao
		if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &reacoes) != nil || len(reacoes) != 1 || reacoes[0].Total != 2 || !reacoes[0].Minha {
			t.Fatalf("contagem: %d %s", resposta.Code, resposta.Body)
		}
		if resposta = chamar(t, ana, http.MethodPut, caminho, `{"emoji":"😂","ativa":false}`); resposta.Code != http.StatusNoContent {
			t.Fatalf("remover: %d %s", resposta.Code, resposta.Body)
		}
		resposta = chamar(t, ana, http.MethodGet, caminho, "")
		if json.Unmarshal(resposta.Body.Bytes(), &reacoes) != nil || len(reacoes) != 1 || reacoes[0].Total != 1 || reacoes[0].Minha {
			t.Fatalf("removida: %d %s", resposta.Code, resposta.Body)
		}
		if resposta = chamar(t, ana, http.MethodPut, caminho, `{"emoji":"invalido","ativa":true}`); resposta.Code != http.StatusBadRequest {
			t.Fatalf("emoji inválido: %d", resposta.Code)
		}
	}
	if _, erro := db.Exec("delete from seguidores where usuario_id = ? and seguidor_id = ?", autor, ana); erro != nil {
		t.Fatal(erro)
	}
	for _, caminho := range []string{fmt.Sprintf("/publicacoes/%d/reacoes", publicacao.ID), fmt.Sprintf("/comentarios/%d/reacoes", comentario)} {
		for _, metodo := range []string{http.MethodGet, http.MethodPut} {
			if resposta := chamar(t, ana, metodo, caminho, `{"emoji":"👍","ativa":true}`); resposta.Code != http.StatusNotFound {
				t.Fatalf("acesso privado: %s %d", metodo, resposta.Code)
			}
		}
	}
	if _, erro := db.Exec("insert into publicacoes_felas (publicacao_id) values (?)", publicacao.ID); erro != nil {
		t.Fatal(erro)
	}
	caminho := fmt.Sprintf("/publicacoes/%d/reacoes", publicacao.ID)
	if resposta := chamar(t, beto, http.MethodGet, caminho, ""); resposta.Code != http.StatusNotFound {
		t.Fatalf("reação exposta para quem não é Fela: %d", resposta.Code)
	}
	if _, erro := db.Exec("insert into felas (dono_id, membro_id) values (?, ?)", autor, beto); erro != nil {
		t.Fatal(erro)
	}
	if resposta := chamar(t, beto, http.MethodGet, caminho, ""); resposta.Code != http.StatusOK {
		t.Fatalf("Fela sem acesso às reações: %d", resposta.Code)
	}
}
