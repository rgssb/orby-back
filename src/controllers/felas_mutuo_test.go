package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/testes"
	"testing"
)

func TestFelasExigeSeguimentoMutuoParaAdicionar(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	pessoa := testes.CriarUsuario(t, db, "pessoa", false)
	caminho := fmt.Sprintf("/felas/%d", pessoa)
	verificarEstado := func(fela, podeAdicionar bool) {
		t.Helper()
		r := chamar(t, dono, http.MethodGet, caminho, "")
		var estado struct {
			Fela          bool `json:"fela"`
			PodeAdicionar bool `json:"podeAdicionar"`
		}
		if r.Code != http.StatusOK || json.Unmarshal(r.Body.Bytes(), &estado) != nil || estado.Fela != fela || estado.PodeAdicionar != podeAdicionar {
			t.Fatalf("estado esperado %t/%t: %d %s", fela, podeAdicionar, r.Code, r.Body.String())
		}
	}
	verificarEstado(false, false)
	for _, par := range [][2]uint64{{dono, pessoa}, {pessoa, dono}} {
		if r := chamar(t, dono, http.MethodPut, caminho, ""); r.Code != http.StatusBadRequest {
			t.Fatalf("adicionou sem seguir mutuamente: %d %s", r.Code, r.Body.String())
		}
		if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", par[0], par[1]); erro != nil {
			t.Fatal(erro)
		}
	}
	verificarEstado(false, true)
	if r := chamar(t, dono, http.MethodPut, caminho, ""); r.Code != http.StatusNoContent {
		t.Fatalf("não adicionou com seguimento mútuo: %d %s", r.Code, r.Body.String())
	}
	if _, erro := db.Exec("delete from seguidores where usuario_id = ? and seguidor_id = ?", dono, pessoa); erro != nil {
		t.Fatal(erro)
	}
	verificarEstado(true, false)
	if r := chamar(t, dono, http.MethodDelete, caminho, ""); r.Code != http.StatusNoContent {
		t.Fatalf("não removeu membro existente: %d %s", r.Code, r.Body.String())
	}
	verificarEstado(false, false)
}
