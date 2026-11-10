package controllers_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"orby/api/src/midia"
	"orby/api/src/seguranca"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestConfiguracoesEPausa(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	outro := testes.CriarUsuario(t, db, "outro", false)
	terceiro := testes.CriarUsuario(t, db, "terceiro", false)
	for _, rota := range []string{"/configuracoes/acessibilidade", "/configuracoes/pessoas-restritas", "/notificacoes/pausa", "/privacidade/notas-felas"} {
		if r := chamar(t, 0, http.MethodGet, rota, ""); r.Code != http.StatusUnauthorized {
			t.Fatalf("%s sem login: %d", rota, r.Code)
		}
	}
	if r := chamar(t, dono, http.MethodPut, "/configuracoes/acessibilidade", `{"tamanho":"grande","reduzirAnimacoes":true}`); r.Code != http.StatusNoContent {
		t.Fatalf("acessibilidade: %d %s", r.Code, r.Body)
	}
	if r := chamar(t, dono, http.MethodGet, "/configuracoes/acessibilidade", ""); !strings.Contains(r.Body.String(), `"tamanho":"grande"`) || !strings.Contains(r.Body.String(), `"reduzirAnimacoes":true`) {
		t.Fatalf("preferência salva: %d %s", r.Code, r.Body)
	}
	if r := chamar(t, outro, http.MethodGet, "/configuracoes/acessibilidade", ""); !strings.Contains(r.Body.String(), `"tamanho":"normal"`) {
		t.Fatalf("preferência vazou: %d %s", r.Code, r.Body)
	}
	if r := chamar(t, dono, http.MethodPut, "/configuracoes/acessibilidade", `{"tamanho":"invalido","reduzirAnimacoes":true}`); r.Code != http.StatusBadRequest {
		t.Fatalf("valor inválido: %d", r.Code)
	}
	if _, err := db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?), (?, ?)", dono, outro, terceiro, dono); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("insert into silenciados (silenciador_id, silenciado_id) values (?, ?)", dono, terceiro); err != nil {
		t.Fatal(err)
	}
	r := chamar(t, dono, http.MethodGet, "/configuracoes/pessoas-restritas", "")
	if !strings.Contains(r.Body.String(), `"bloqueados":[{"id":2`) || !strings.Contains(r.Body.String(), `"silenciados":[{"id":3`) {
		t.Fatalf("pessoas restritas: %d %s", r.Code, r.Body)
	}
	if r = chamar(t, dono, http.MethodPost, fmt.Sprintf("/usuarios/%d/desbloquear", outro), ""); r.Code != http.StatusNoContent {
		t.Fatalf("desbloquear: %d %s", r.Code, r.Body)
	}
	if r = chamar(t, dono, http.MethodGet, "/configuracoes/pessoas-restritas", ""); !strings.Contains(r.Body.String(), `"bloqueados":[]`) {
		t.Fatalf("lista após desbloqueio: %s", r.Body)
	}
	if _, err := db.Exec("insert into notificacoes (usuario_id, ator_id, tipo) values (?, ?, 'seguidor'), (?, ?, 'pedido_seguir')", dono, outro, dono, terceiro); err != nil {
		t.Fatal(err)
	}
	if r = chamar(t, dono, http.MethodPut, "/notificacoes/pausa", `{"duracao":"permanente"}`); r.Code != http.StatusNoContent {
		t.Fatalf("pausar: %d %s", r.Code, r.Body)
	}
	if r = chamar(t, dono, http.MethodGet, "/notificacoes/nao-lidas", ""); !strings.Contains(r.Body.String(), `"total":1`) {
		t.Fatalf("pausa deve preservar pedido: %s", r.Body)
	}
	if r = chamar(t, dono, http.MethodPost, "/notificacoes/lidas", ""); r.Code != http.StatusNoContent {
		t.Fatalf("marcar lidas: %d", r.Code)
	}
	if r = chamar(t, dono, http.MethodPut, "/notificacoes/pausa", `{"duracao":"retomar"}`); r.Code != http.StatusNoContent {
		t.Fatal(r.Code)
	}
	if r = chamar(t, dono, http.MethodGet, "/notificacoes/nao-lidas", ""); !strings.Contains(r.Body.String(), `"total":1`) {
		t.Fatalf("aviso oculto deve continuar não lido: %s", r.Body)
	}
	if r = chamar(t, dono, http.MethodPut, "/notificacoes/pausa", `{"duracao":"1h"}`); r.Code != http.StatusNoContent {
		t.Fatal(r.Code)
	}
	if r = chamar(t, dono, http.MethodGet, "/notificacoes/pausa", ""); !strings.Contains(r.Body.String(), `"permanente":false`) || strings.Contains(r.Body.String(), `"segundosRestantes":0`) {
		t.Fatalf("pausa temporária: %d %s", r.Code, r.Body)
	}
	if r = chamar(t, dono, http.MethodPut, "/notificacoes/pausa", `{"duracao":"42h"}`); r.Code != http.StatusBadRequest {
		t.Fatalf("duração inválida: %d", r.Code)
	}
	if _, err := db.Exec("update preferencias_configuracoes set pausa_ate = date_sub(utc_timestamp(), interval 1 second) where usuario_id = ?", dono); err != nil {
		t.Fatal(err)
	}
	if r = chamar(t, dono, http.MethodGet, "/notificacoes/nao-lidas", ""); !strings.Contains(r.Body.String(), `"total":1`) {
		t.Fatalf("pausa expirada: %s", r.Body)
	}
	if r = chamar(t, dono, http.MethodPut, "/conta/email", `{"email":"novo@orby.test"}`); r.Code != http.StatusNoContent {
		t.Fatalf("email: %d %s", r.Code, r.Body)
	}
	if r = chamar(t, dono, http.MethodPut, "/perfil/dados", `{"nome":"Dono novo","nick":"dono","bio":"bio"}`); r.Code != http.StatusNoContent {
		t.Fatalf("perfil: %d %s", r.Code, r.Body)
	}
	var email string
	if err := db.QueryRow("select email from usuarios where id = ?", dono).Scan(&email); err != nil || email != "novo@orby.test" {
		t.Fatalf("editar perfil alterou e-mail: %s %v", email, err)
	}
}

func TestExportacaoProtegeDadosEMidias(t *testing.T) {
	db := testes.NovoBanco(t)
	dono := testes.CriarUsuario(t, db, "dono", false)
	outro := testes.CriarUsuario(t, db, "outro", false)
	terceiro := testes.CriarUsuario(t, db, "terceiro", false)
	senha, err := seguranca.Hash("senha-segura")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("update usuarios set senha = ? where id = ?", senha, dono); err != nil {
		t.Fatal(err)
	}
	post := criarPublicacao(t, db, dono)
	criarPublicacao(t, db, terceiro)
	if _, err = db.Exec("insert into imagens_publicacoes (publicacao_id,tipo,descricao) values (?, 'image/png', 'imagem minha')", post); err != nil {
		t.Fatal(err)
	}
	if err = midia.Gravar(midia.Publicacoes, 1, []byte("imagem-publicacao")); err != nil {
		t.Fatal(err)
	}
	res, err := db.Exec("insert into conversas (usuario_menor,usuario_maior,iniciada_por,aceita) values (?, ?, ?, true)", dono, outro, dono)
	if err != nil {
		t.Fatal(err)
	}
	conversa, _ := res.LastInsertId()
	if _, err = db.Exec("insert into mensagens (conversa_id, remetente_id, texto) values (?, ?, 'apagada da minha visão'), (?, ?, 'mensagem recebida')", conversa, dono, conversa, outro); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("update conversas set limpa_menor = 1 where id = ?", conversa); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("insert into imagens_mensagens (mensagem_id,tipo) values (2,'image/jpeg')"); err != nil {
		t.Fatal(err)
	}
	if err = midia.Gravar(midia.Mensagens, 2, []byte("imagem-mensagem")); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`insert into grupos (nome,criador_id) values ('grupo permitido',?), ('grupo alheio',?)`, dono, terceiro); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`insert into grupo_membros (grupo_id,usuario_id) values (1,?),(1,?),(2,?)`, dono, outro, terceiro); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`insert into grupo_mensagens (grupo_id,remetente_id,texto) values (1,?,'mensagem grupo permitida'),(2,?,'mensagem grupo secreta')`, outro, terceiro); err != nil {
		t.Fatal(err)
	}
	if r := chamar(t, dono, http.MethodPost, "/conta/exportar", `{"senha":"errada"}`); r.Code != http.StatusForbidden {
		t.Fatalf("senha incorreta: %d", r.Code)
	}
	r := chamar(t, dono, http.MethodPost, "/conta/exportar", `{"senha":"senha-segura"}`)
	if r.Code != http.StatusOK {
		t.Fatalf("exportação: %d %s", r.Code, r.Body)
	}
	leitor, err := zip.NewReader(bytes.NewReader(r.Body.Bytes()), int64(r.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	arquivos := map[string]string{}
	for _, f := range leitor.File {
		entrada, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		conteudo, err := io.ReadAll(entrada)
		entrada.Close()
		if err != nil {
			t.Fatal(err)
		}
		arquivos[f.Name] = string(conteudo)
	}
	if !strings.Contains(arquivos["mensagens.json"], "mensagem recebida") || strings.Contains(arquivos["mensagens.json"], "apagada da minha visão") {
		t.Fatalf("mensagens incorretas: %s", arquivos["mensagens.json"])
	}
	if arquivos["midias/publicacoes/1.png"] != "imagem-publicacao" || arquivos["midias/mensagens/2.jpg"] != "imagem-mensagem" {
		t.Fatalf("mídias ausentes: %v", arquivos)
	}
	if !strings.Contains(arquivos["mensagens_grupos.json"], "mensagem grupo permitida") || strings.Contains(arquivos["mensagens_grupos.json"], "mensagem grupo secreta") {
		t.Fatalf("mensagens de grupos incorretas: %s", arquivos["mensagens_grupos.json"])
	}
	if strings.Contains(arquivos["conta.json"], "senha") || strings.Contains(arquivos["conta.json"], "terceiro") ||
		strings.Count(arquivos["publicacoes.json"], "conteudo secreto") != 1 {
		t.Fatalf("exportação incluiu dados indevidos")
	}
	if err = midia.Remover(midia.Mensagens, 2); err != nil {
		t.Fatal(err)
	}
	if r = chamar(t, dono, http.MethodPost, "/conta/exportar", `{"senha":"senha-segura"}`); r.Code != http.StatusInternalServerError || strings.HasPrefix(r.Body.String(), "PK") {
		t.Fatalf("mídia ausente não deve produzir arquivo incompleto: %d", r.Code)
	}
}
