package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/repositorios"
	"orby/api/src/testes"
	"testing"
)

func TestNotasFelasPermissoesEPrivacidade(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", true)
	autor := testes.CriarUsuario(t, db, "autor", false)
	outro := testes.CriarUsuario(t, db, "outro", false)
	rota := fmt.Sprintf("/usuarios/%d/notas-felas", dono)
	if _, err := db.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?)", dono, autor); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("insert into felas (dono_id, membro_id) values (?, ?)", dono, autor); err != nil {
		t.Fatal(err)
	}
	verificar := func(usuario uint64, metodo, caminho, corpo string, esperado int) {
		t.Helper()
		if resp := chamar(t, usuario, metodo, caminho, corpo); resp.Code != esperado {
			t.Fatalf("%s %s usuario %d: %d %s", metodo, caminho, usuario, resp.Code, resp.Body)
		}
	}
	verificar(outro, "GET", rota, "", http.StatusNotFound)
	verificar(outro, "POST", rota, `{"conteudo":"invadindo"}`, http.StatusNotFound)
	verificar(dono, "POST", rota, `{"conteudo":"própria"}`, http.StatusNotFound)
	verificar(autor, "POST", rota, `{"conteudo":"   "}`, http.StatusBadRequest)
	verificar(autor, "POST", rota, `{"conteudo":"primeira"}`, http.StatusNoContent)
	verificar(autor, "POST", rota, `{"conteudo":"segunda"}`, http.StatusNoContent)
	resposta := chamar(t, autor, "GET", rota, "")
	var dados repositorios.NotasFelas
	if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &dados) != nil || len(dados.Notas) != 2 || !dados.PodeEscrever {
		t.Fatalf("notas do fela: %d %s", resposta.Code, resposta.Body)
	}
	nota1, nota2 := dados.Notas[0].ID, dados.Notas[1].ID
	verificar(outro, "DELETE", fmt.Sprintf("/notas-felas/%d", nota1), "", http.StatusNotFound)
	verificar(dono, "PUT", "/privacidade/notas-felas", `{"ativas":false}`, http.StatusNoContent)
	verificar(autor, "POST", rota, `{"conteudo":"oculta"}`, http.StatusNotFound)
	resposta = chamar(t, autor, "GET", rota, "")
	if json.Unmarshal(resposta.Body.Bytes(), &dados) != nil || dados.Ativas || len(dados.Notas) != 0 {
		t.Fatalf("notas desativadas: %s", resposta.Body)
	}
	verificar(dono, "PUT", "/privacidade/notas-felas", `{"ativas":true}`, http.StatusNoContent)
	resposta = chamar(t, dono, "GET", rota, "")
	if json.Unmarshal(resposta.Body.Bytes(), &dados) != nil || len(dados.Notas) != 2 {
		t.Fatalf("notas não foram preservadas: %s", resposta.Body)
	}
	verificar(autor, "DELETE", fmt.Sprintf("/notas-felas/%d", nota1), "", http.StatusNoContent)
	verificar(dono, "DELETE", fmt.Sprintf("/notas-felas/%d", nota2), "", http.StatusNoContent)
	if _, err := db.Exec("delete from felas where dono_id = ? and membro_id = ?", dono, autor); err != nil {
		t.Fatal(err)
	}
	verificar(autor, "POST", rota, `{"conteudo":"após remoção"}`, http.StatusNotFound)
	if _, err := db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", dono, autor); err != nil {
		t.Fatal(err)
	}
	verificar(autor, "GET", rota, "", http.StatusNotFound)
}

func TestNotasFelasPaginacaoEBloqueioDeAutores(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	autor := testes.CriarUsuario(t, db, "autor", false)
	visitante := testes.CriarUsuario(t, db, "visitante", false)
	if _, err := db.Exec("insert into felas (dono_id, membro_id) values (?, ?)", dono, autor); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 21; i++ {
		if err := repositorios.CriarNotaFela(db, dono, autor, "nota"); err != nil {
			t.Fatal(err)
		}
	}
	primeira, err := repositorios.ListarNotasFelas(db, dono, visitante, 0)
	if err != nil || len(primeira.Notas) != 20 || primeira.Proxima == 0 {
		t.Fatalf("primeira página: %+v %v", primeira, err)
	}
	segunda, err := repositorios.ListarNotasFelas(db, dono, visitante, primeira.Proxima)
	if err != nil || len(segunda.Notas) != 1 || segunda.Proxima != 0 {
		t.Fatalf("segunda página: %+v %v", segunda, err)
	}
	if _, err := db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", visitante, autor); err != nil {
		t.Fatal(err)
	}
	ocultas, err := repositorios.ListarNotasFelas(db, dono, visitante, 0)
	if err != nil || len(ocultas.Notas) != 0 {
		t.Fatalf("autor bloqueado: %+v %v", ocultas, err)
	}
	if _, err := db.Exec("delete from bloqueios where bloqueador_id = ?", visitante); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", dono, autor); err != nil {
		t.Fatal(err)
	}
	ocultas, err = repositorios.ListarNotasFelas(db, dono, visitante, 0)
	if err != nil || len(ocultas.Notas) != 0 {
		t.Fatalf("autor bloqueado pelo dono: %+v %v", ocultas, err)
	}
	visiveis, err := repositorios.ListarNotasFelas(db, dono, dono, 0)
	if err != nil || len(visiveis.Notas) != 20 || !visiveis.Notas[0].PodeExcluir {
		t.Fatalf("dono deve poder remover as notas: %+v %v", visiveis, err)
	}
}
