package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/repositorios"
	"orby/api/src/testes"
	"testing"
)

func TestCriarPastaAoSalvarPublicacao(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	outro := testes.CriarUsuario(t, db, "outro", false)
	post := criarPublicacao(t, db, dono)
	rota := "/publicacoes/" + post + "/pastas"
	if resposta := chamar(t, dono, http.MethodPost, rota, `{"nome":"sem salvo"}`); resposta.Code != http.StatusNotFound {
		t.Fatalf("publicação não salva: %d %s", resposta.Code, resposta.Body)
	}
	if resposta := chamar(t, dono, http.MethodPost, "/publicacoes/"+post+"/salvar", ""); resposta.Code != http.StatusNoContent {
		t.Fatal(resposta.Body)
	}
	for _, nome := range []string{"primeira", "segunda"} {
		resposta := chamar(t, dono, http.MethodPost, rota, fmt.Sprintf(`{"nome":%q}`, nome))
		if resposta.Code != http.StatusCreated {
			t.Fatalf("criação: %d %s", resposta.Code, resposta.Body)
		}
	}
	resposta := chamar(t, dono, http.MethodGet, rota, "")
	var pastas []repositorios.PastaSalvos
	if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &pastas) != nil || len(pastas) != 2 || !pastas[0].Incluida || !pastas[1].Incluida {
		t.Fatalf("pastas existentes devem continuar associadas: %d %s", resposta.Code, resposta.Body)
	}
	if _, erro := db.Exec("insert into salvos (usuario_id, publicacao_id) values (?, ?)", outro, post); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := db.Exec("update usuarios set privado = 1 where id = ?", dono); erro != nil {
		t.Fatal(erro)
	}
	if resposta := chamar(t, outro, http.MethodPost, rota, `{"nome":"sem acesso"}`); resposta.Code != http.StatusNotFound {
		t.Fatalf("publicação invisível: %d %s", resposta.Code, resposta.Body)
	}
	var quantidade int
	if erro := db.QueryRow("select count(*) from pastas_salvos where usuario_id = ?", outro).Scan(&quantidade); erro != nil || quantidade != 0 {
		t.Fatalf("não deve criar pasta na falha: %d %v", quantidade, erro)
	}
}
