package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/repositorios"
	"orby/api/src/testes"
	"testing"
)

func TestCurtidasPerfilPrivacidadeEVisibilidade(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	visitante := testes.CriarUsuario(t, db, "visitante", false)
	privado := testes.CriarUsuario(t, db, "privado", true)
	publico := testes.CriarUsuario(t, db, "publico", false)
	p1, p2, p3 := criarPublicacao(t, db, publico), criarPublicacao(t, db, privado), criarPublicacao(t, db, publico)
	proprio := criarPublicacao(t, db, dono)
	for _, post := range []string{p1, p2, p3} {
		if _, erro := db.Exec("insert into curtidas_publicacoes (publicacao_id, usuario_id) values (?, ?)", post, dono); erro != nil {
			t.Fatal(erro)
		}
	}
	if _, erro := db.Exec("insert into curtidas_publicacoes (publicacao_id, usuario_id) values (?, ?)", proprio, visitante); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := db.Exec("insert into publicacoes_felas (publicacao_id) values (?)", p3); erro != nil {
		t.Fatal(erro)
	}
	rota := fmt.Sprintf("/usuarios/%d/curtidas?lista=1", dono)
	consultar := func(usuario uint64) repositorios.CurtidasPerfil {
		t.Helper()
		resposta := chamar(t, usuario, http.MethodGet, rota, "")
		var resultado repositorios.CurtidasPerfil
		if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &resultado) != nil {
			t.Fatalf("consulta: %d %s", resposta.Code, resposta.Body)
		}
		return resultado
	}
	if r := consultar(visitante); r.PodeVer || r.ListaPublica || !r.TotalVisivel || len(r.Publicacoes) != 0 || r.Total != 1 {
		t.Fatalf("lista deve ser privada por padrão: %+v", r)
	}
	if r := consultar(dono); !r.PodeVer || r.ListaPublica || !r.TotalVisivel || len(r.Publicacoes) != 1 || r.Total != 1 {
		t.Fatalf("dono deve ver apenas publicações visíveis a ele: %+v", r)
	}
	if resposta := chamar(t, dono, http.MethodGet, "/privacidade/curtidas", ""); resposta.Code != http.StatusOK || resposta.Body.String() != "{\"publica\":false}\n" {
		t.Fatalf("preferência inicial: %d %s", resposta.Code, resposta.Body)
	}
	if resposta := chamar(t, dono, http.MethodPut, "/privacidade/curtidas", `{}`); resposta.Code != http.StatusBadRequest {
		t.Fatalf("preferência inválida: %d %s", resposta.Code, resposta.Body)
	}
	if resposta := chamar(t, dono, http.MethodPut, "/privacidade/curtidas", `{"publica":true}`); resposta.Code != http.StatusNoContent {
		t.Fatalf("alterar privacidade: %d %s", resposta.Code, resposta.Body)
	}
	if r := consultar(visitante); !r.PodeVer || !r.ListaPublica || !r.TotalVisivel || len(r.Publicacoes) != 1 || fmt.Sprint(r.Publicacoes[0].ID) != p1 {
		t.Fatalf("não deve mostrar posts privados nem Felas: %+v", r)
	}
	if resposta := chamar(t, dono, http.MethodPut, "/privacidade/curtidas", `{"publica":false}`); resposta.Code != http.StatusNoContent {
		t.Fatalf("privatizar novamente: %d %s", resposta.Code, resposta.Body)
	}
	if r := consultar(visitante); r.PodeVer || len(r.Publicacoes) != 0 {
		t.Fatalf("lista deve voltar a ser privada: %+v", r)
	}
	if resposta := chamar(t, dono, http.MethodPut, "/privacidade/curtidas", `{"publica":true}`); resposta.Code != http.StatusNoContent {
		t.Fatalf("abrir novamente: %d %s", resposta.Code, resposta.Body)
	}
	if _, erro := db.Exec("update usuarios set privado = 1 where id = ?", dono); erro != nil {
		t.Fatal(erro)
	}
	if r := consultar(visitante); r.PodeVer || r.ListaPublica || r.TotalVisivel || r.Total != 0 || len(r.Publicacoes) != 0 {
		t.Fatalf("perfil privado deve prevalecer: %+v", r)
	}
	if _, erro := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", dono, visitante); erro != nil {
		t.Fatal(erro)
	}
	if r := consultar(visitante); !r.PodeVer || len(r.Publicacoes) != 1 || r.Total != 1 {
		t.Fatalf("seguidor deveria ter acesso: %+v", r)
	}
	if _, erro := db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", dono, visitante); erro != nil {
		t.Fatal(erro)
	}
	if r := consultar(visitante); r.PodeVer || r.Total != 0 {
		t.Fatalf("bloqueio deve prevalecer: %+v", r)
	}
}
