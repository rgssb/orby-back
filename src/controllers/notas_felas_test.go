package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/eventos"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/testes"
	"testing"
)

func TestNotaEnviaNotificacaoAoDonoEmTempoReal(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	autor := testes.CriarUsuario(t, db, "autor", false)
	outro := testes.CriarUsuario(t, db, "outro", false)
	if _, err := db.Exec("insert into felas (dono_id, membro_id) values (?, ?)", dono, autor); err != nil {
		t.Fatal(err)
	}
	canal, cancelar := eventos.Assinar(dono)
	defer cancelar()
	rota := fmt.Sprintf("/usuarios/%d/notas-felas", dono)
	if r := chamar(t, outro, http.MethodPost, rota, `{"conteudo":"não autorizado"}`); r.Code != http.StatusNotFound {
		t.Fatalf("autor não autorizado: %d", r.Code)
	}
	select {
	case e := <-canal:
		t.Fatalf("evento indevido: %+v", e)
	default:
	}
	for _, conteudo := range []string{"Um recado", "Outro recado"} {
		if r := chamar(t, autor, http.MethodPost, rota, fmt.Sprintf(`{"conteudo":%q}`, conteudo)); r.Code != http.StatusNoContent {
			t.Fatalf("criação: %d %s", r.Code, r.Body)
		}
		select {
		case e := <-canal:
			if e.Tipo != "notificacao" {
				t.Fatalf("evento inesperado: %+v", e)
			}
		default:
			t.Fatal("notificação em tempo real não enviada")
		}
	}
	r := chamar(t, dono, http.MethodGet, "/notificacoes", "")
	var lista []modelos.Notificacao
	if err := json.Unmarshal(r.Body.Bytes(), &lista); err != nil || len(lista) != 2 {
		t.Fatalf("notificações: %s (%v)", r.Body, err)
	}
	for _, n := range lista {
		if n.Tipo != modelos.NotificacaoNotaFela || n.AtorID != autor || n.PublicacaoID != 0 || n.Lida {
			t.Fatalf("notificação incorreta: %+v", n)
		}
	}
	if r := chamar(t, outro, http.MethodGet, "/notificacoes", ""); r.Code != http.StatusOK || r.Body.String() != "[]\n" {
		t.Fatalf("outro usuário não deve receber: %d %s", r.Code, r.Body)
	}
	if r := chamar(t, dono, http.MethodPut, "/notificacoes/preferencias", `{"silenciados":["nota_fela"]}`); r.Code != http.StatusNoContent {
		t.Fatalf("silenciar: %d %s", r.Code, r.Body)
	}
	if r := chamar(t, dono, http.MethodGet, "/notificacoes/nao-lidas", ""); r.Body.String() != "{\"total\":0}\n" {
		t.Fatalf("notificações silenciadas: %s", r.Body)
	}
	if r := chamar(t, dono, http.MethodPut, "/notificacoes/preferencias", `{"silenciados":[]}`); r.Code != http.StatusNoContent {
		t.Fatalf("reativar: %d %s", r.Code, r.Body)
	}
	if r := chamar(t, dono, http.MethodGet, "/notificacoes/nao-lidas", ""); r.Body.String() != "{\"total\":2}\n" {
		t.Fatalf("notificações devem ser preservadas: %s", r.Body)
	}
}

func TestNotaNaoEGravadaSeNotificacaoFalhar(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	autor := testes.CriarUsuario(t, db, "autor", false)
	if _, err := db.Exec("insert into felas (dono_id, membro_id) values (?, ?)", dono, autor); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("alter table notificacoes add constraint teste_sem_nota check (tipo <> 'nota_fela')"); err != nil {
		t.Fatal(err)
	}
	canal, cancelar := eventos.Assinar(dono)
	defer cancelar()
	rota := fmt.Sprintf("/usuarios/%d/notas-felas", dono)
	if r := chamar(t, autor, http.MethodPost, rota, `{"conteudo":"recado"}`); r.Code != http.StatusInternalServerError {
		t.Fatalf("falha deveria cancelar a criação: %d %s", r.Code, r.Body)
	}
	var quantidade int
	if err := db.QueryRow("select count(*) from notas_felas").Scan(&quantidade); err != nil || quantidade != 0 {
		t.Fatalf("recado não deveria ser gravado: %d %v", quantidade, err)
	}
	select {
	case e := <-canal:
		t.Fatalf("evento após rollback: %+v", e)
	default:
	}
}

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
