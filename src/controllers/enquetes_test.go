package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
	"time"
)

func TestEnqueteVotoUnicoResultadosEPrazo(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	criar := chamar(t, autor, http.MethodPost, "/publicacoes", `{"conteudo":"Pergunta","enquete":{"opcoes":[{"texto":"Sim"},{"texto":"Não"}]}}`)
	if criar.Code != http.StatusCreated {
		t.Fatalf("criação: %d %s", criar.Code, criar.Body)
	}
	var criada modelos.Publicacao
	if erro := json.Unmarshal(criar.Body.Bytes(), &criada); erro != nil {
		t.Fatal(erro)
	}
	caminho := fmt.Sprintf("/publicacoes/%d", criada.ID)
	buscar := func(usuario uint64) modelos.Publicacao {
		t.Helper()
		resposta := chamar(t, usuario, http.MethodGet, caminho, "")
		if resposta.Code != http.StatusOK {
			t.Fatalf("buscar: %d %s", resposta.Code, resposta.Body)
		}
		var post modelos.Publicacao
		if erro := json.Unmarshal(resposta.Body.Bytes(), &post); erro != nil || post.Enquete == nil {
			t.Fatalf("enquete ausente: %s (%v)", resposta.Body, erro)
		}
		return post
	}
	post := buscar(ana)
	if len(post.Enquete.Opcoes) != 2 || post.Enquete.Votou {
		t.Fatalf("enquete inicial: %+v", post.Enquete)
	}
	votar := func(usuario, opcao uint64) int {
		return chamar(t, usuario, http.MethodPost, fmt.Sprintf("%s/enquete/%d/votar", caminho, opcao), "").Code
	}
	if status := votar(ana, post.Enquete.Opcoes[0].ID); status != http.StatusNoContent {
		t.Fatalf("voto: %d", status)
	}
	if status := votar(ana, post.Enquete.Opcoes[1].ID); status != http.StatusConflict {
		t.Fatalf("troca de voto permitida: %d", status)
	}
	postAna := buscar(ana)
	if !postAna.Enquete.Votou || postAna.Enquete.TotalVotos != 1 || !postAna.Enquete.Opcoes[0].Minha || postAna.Enquete.Opcoes[0].Votos != 1 {
		t.Fatalf("resultado de quem votou: %+v", postAna.Enquete)
	}
	postBeto := buscar(beto)
	if postBeto.Enquete.Votou || postBeto.Enquete.TotalVotos != 0 || postBeto.Enquete.Opcoes[0].Votos != 0 {
		t.Fatalf("resultados expostos antes do voto: %+v", postBeto.Enquete)
	}
	if _, erro := db.Exec("update enquetes set encerra_em = ? where publicacao_id = ?", time.Now().Add(-time.Minute).Unix(), criada.ID); erro != nil {
		t.Fatal(erro)
	}
	if status := votar(beto, post.Enquete.Opcoes[1].ID); status != http.StatusConflict {
		t.Fatalf("voto após encerramento: %d", status)
	}
	if fechada := buscar(beto).Enquete; !fechada.Encerrada || fechada.TotalVotos != 1 {
		t.Fatalf("resultados após encerramento: %+v", fechada)
	}
}

func TestEnqueteValidacaoEPrivacidade(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", true)
	outro := testes.CriarUsuario(t, db, "outro", false)
	for _, corpo := range []string{
		`{"conteudo":"pergunta","enquete":{"opcoes":[{"texto":"só"}]}}`,
		`{"conteudo":"pergunta","enquete":{"opcoes":[{"texto":"igual"},{"texto":" Igual "}]}}`,
		`{"conteudo":"pergunta","enquete":{"opcoes":[{"texto":"a"},{"texto":"b"}],"encerraEm":"2000-01-01T00:00:00Z"}}`,
	} {
		if resposta := chamar(t, autor, http.MethodPost, "/publicacoes", corpo); resposta.Code != http.StatusBadRequest {
			t.Fatalf("enquete inválida aceita: %d %s", resposta.Code, resposta.Body)
		}

	}
	var total int
	if erro := db.QueryRow("select count(*) from publicacoes").Scan(&total); erro != nil || total != 0 {
		t.Fatalf("publicações inválidas foram gravadas: %d %v", total, erro)
	}
	criar := chamar(t, autor, http.MethodPost, "/publicacoes", `{"conteudo":"segredo","enquete":{"opcoes":[{"texto":"a"},{"texto":"b"}]}}`)
	var post modelos.Publicacao
	if criar.Code != http.StatusCreated || json.Unmarshal(criar.Body.Bytes(), &post) != nil {
		t.Fatalf("criar: %d %s", criar.Code, criar.Body)
	}
	var opcao uint64
	if erro := db.QueryRow("select id from opcoes_enquetes where publicacao_id = ? limit 1", post.ID).Scan(&opcao); erro != nil {
		t.Fatal(erro)
	}
	caminho := fmt.Sprintf("/publicacoes/%d/enquete/%d/votar", post.ID, opcao)
	if resposta := chamar(t, outro, http.MethodPost, caminho, ""); resposta.Code != http.StatusNotFound {
		t.Fatalf("voto em publicação privada: %d", resposta.Code)
	}
}
