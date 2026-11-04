package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/testes"
	"testing"
)

func TestPastasPrivadasMultiplasEIntegridade(t *testing.T) {
	db := testes.NovoBanco(t)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	post := criarPublicacao(t, db, ana)
	rotaPost := "/publicacoes/" + post
	if resposta := chamar(t, ana, http.MethodPost, rotaPost+"/salvar", ""); resposta.Code != http.StatusNoContent {
		t.Fatal(resposta.Body)
	}

	criarPasta := func(usuario uint64, nome string) uint64 {
		t.Helper()
		resposta := chamar(t, usuario, http.MethodPost, "/pastas", fmt.Sprintf(`{"nome":%q}`, nome))
		var pasta repositorios.PastaSalvos
		if resposta.Code != http.StatusCreated || json.Unmarshal(resposta.Body.Bytes(), &pasta) != nil {
			t.Fatalf("criação da pasta: %d %s", resposta.Code, resposta.Body)
		}
		return pasta.ID
	}
	primeira, segunda, alheia := criarPasta(ana, "primeira"), criarPasta(ana, "segunda"), criarPasta(beto, "privada")
	associar := func(ids string) int {
		t.Helper()
		resposta := chamar(t, ana, http.MethodPut, rotaPost+"/pastas", ids)
		return resposta.Code
	}
	if status := associar(fmt.Sprintf(`{"pastas":[%d,%d]}`, primeira, segunda)); status != http.StatusNoContent {
		t.Fatalf("associação: %d", status)
	}
	if status := associar(fmt.Sprintf(`{"pastas":[%d,%d]}`, primeira, alheia)); status != http.StatusNotFound {
		t.Fatalf("pasta de outra pessoa foi aceita: %d", status)
	}
	resposta := chamar(t, ana, http.MethodGet, rotaPost+"/pastas", "")
	var pastas []repositorios.PastaSalvos
	if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &pastas) != nil || len(pastas) != 2 || !pastas[0].Incluida || !pastas[1].Incluida {
		t.Fatalf("associações deveriam permanecer após falha: %s", resposta.Body)
	}
	for _, id := range []uint64{primeira, segunda} {
		resposta = chamar(t, ana, http.MethodGet, fmt.Sprintf("/publicacoes/salvas?pasta=%d", id), "")
		var salvos []modelos.Publicacao
		if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &salvos) != nil || len(salvos) != 1 {
			t.Fatalf("consulta da pasta %d: %d %s", id, resposta.Code, resposta.Body)
		}
	}
	if resposta = chamar(t, beto, http.MethodGet, fmt.Sprintf("/publicacoes/salvas?pasta=%d", primeira), ""); resposta.Code != http.StatusNotFound {
		t.Fatalf("outra pessoa acessou a pasta: %d", resposta.Code)
	}
	if resposta = chamar(t, beto, http.MethodDelete, fmt.Sprintf("/pastas/%d", primeira), ""); resposta.Code != http.StatusNotFound {
		t.Fatalf("outra pessoa excluiu a pasta: %d", resposta.Code)
	}
	if resposta = chamar(t, ana, http.MethodDelete, fmt.Sprintf("/pastas/%d", primeira), ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("exclusão: %d", resposta.Code)
	}
	var quantidade int
	if erro := db.QueryRow("select count(*) from pastas_publicacoes where usuario_id = ?", ana).Scan(&quantidade); erro != nil || quantidade != 1 {
		t.Fatalf("outras pastas devem permanecer: %d %v", quantidade, erro)
	}
	if resposta = chamar(t, ana, http.MethodDelete, rotaPost+"/salvar", ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("remover salvo: %d", resposta.Code)
	}
	if erro := db.QueryRow("select count(*) from pastas_publicacoes where usuario_id = ?", ana).Scan(&quantidade); erro != nil || quantidade != 0 {
		t.Fatalf("associações deveriam ser removidas: %d %v", quantidade, erro)
	}
}
