package controllers_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"orby/api/src/modelos"
	"orby/api/src/repositorios"
	"orby/api/src/testes"
	"strings"
	"testing"
)

func TestDenunciasPorUsuarioETipo(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "autor", false)
	comentarista := testes.CriarUsuario(t, db, "comentarista", false)
	pessoa := testes.CriarUsuario(t, db, "pessoa", false)
	outra := testes.CriarUsuario(t, db, "outra", false)
	admin := testes.CriarUsuario(t, db, "admin", false)
	if _, erro := db.Exec("update usuarios set admin = 1 where id = ?", admin); erro != nil {
		t.Fatal(erro)
	}
	publicacao := criarPublicacao(t, db, autor)
	resultado, erro := db.Exec("insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, 'comentário')", publicacao, comentarista)
	if erro != nil {
		t.Fatal(erro)
	}
	comentario, _ := resultado.LastInsertId()

	alvos := []struct {
		tipo string
		id   string
		dono uint64
	}{
		{"perfil", fmt.Sprint(autor), autor},
		{"publicacao", publicacao, autor},
		{"comentario", fmt.Sprint(comentario), comentarista},
	}
	for _, alvo := range alvos {
		corpo := fmt.Sprintf(`{"tipo":%q,"alvoId":%s,"motivo":"spam"}`, alvo.tipo, alvo.id)
		resposta := chamar(t, pessoa, http.MethodPost, "/denuncias", corpo)
		var dados struct {
			UsuarioID uint64 `json:"usuarioId"`
		}
		if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &dados) != nil || dados.UsuarioID != alvo.dono {
			t.Fatalf("denúncia de %s: status=%d corpo=%s", alvo.tipo, resposta.Code, resposta.Body)
		}
		if duplicada := chamar(t, pessoa, http.MethodPost, "/denuncias", corpo); duplicada.Code != http.StatusConflict {
			t.Fatalf("denúncia repetida de %s: status=%d", alvo.tipo, duplicada.Code)
		}
	}
	consulta := fmt.Sprintf(`{"alvos":[{"tipo":"perfil","alvoId":%d},{"tipo":"publicacao","alvoId":%s},{"tipo":"comentario","alvoId":%d}]}`, autor, publicacao, comentario)
	if resposta := chamar(t, 0, http.MethodPost, "/denuncias/consulta", consulta); resposta.Code != http.StatusUnauthorized {
		t.Fatalf("consulta sem autenticação: %d", resposta.Code)
	}
	conferir := func(usuario uint64, esperados int) {
		t.Helper()
		resposta := chamar(t, usuario, http.MethodPost, "/denuncias/consulta", consulta)
		var encontrados []modelos.AlvoDeDenuncia
		if resposta.Code != http.StatusOK || json.Unmarshal(resposta.Body.Bytes(), &encontrados) != nil || len(encontrados) != esperados {
			t.Fatalf("consulta: status=%d corpo=%s", resposta.Code, resposta.Body)
		}
	}
	conferir(pessoa, 3)
	conferir(outra, 0)
	if erro := repositorios.NovoRepositorioDeDenuncias(db).Resolver(autor); erro != nil {
		t.Fatal(erro)
	}
	conferir(pessoa, 1)
	corpoPerfil := fmt.Sprintf(`{"tipo":"perfil","alvoId":%d,"motivo":"spam"}`, autor)
	if resposta := chamar(t, pessoa, http.MethodPost, "/denuncias", corpoPerfil); resposta.Code != http.StatusOK {
		t.Fatalf("nova denúncia após arquivamento: %d", resposta.Code)
	}
	if resposta := chamar(t, pessoa, http.MethodPost, "/denuncias", corpoPerfil); resposta.Code != http.StatusConflict {
		t.Fatalf("denúncia aberta repetida: %d", resposta.Code)
	}
	if resposta := chamar(t, outra, http.MethodPost, "/denuncias", corpoPerfil); resposta.Code != http.StatusOK {
		t.Fatalf("outra pessoa deveria poder denunciar: %d", resposta.Code)
	}
	corpoPost := fmt.Sprintf(`{"tipo":"publicacao","alvoId":%s,"motivo":"spam"}`, publicacao)
	if resposta := chamar(t, pessoa, http.MethodPost, "/denuncias", corpoPost); resposta.Code != http.StatusOK {
		t.Fatalf("nova denúncia de publicação após arquivamento: %d", resposta.Code)
	}
	conferir(pessoa, 3)
	caminho := fmt.Sprintf("/admin/usuarios/%d", autor)
	if resposta := chamar(t, admin, http.MethodPost, caminho+"/reativar", ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("reativar conta ativa: %d", resposta.Code)
	}
	conferir(pessoa, 3)
	if resposta := chamar(t, admin, http.MethodPost, caminho+"/suspender", ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("suspender conta: %d", resposta.Code)
	}
	perfil := chamar(t, admin, http.MethodGet, fmt.Sprintf("/usuarios/%d", autor), "")
	if perfil.Code != http.StatusOK || !strings.Contains(perfil.Body.String(), `"suspenso":true`) {
		t.Fatalf("administrador deve ver suspensão: %s", perfil.Body)
	}
	perfil = chamar(t, pessoa, http.MethodGet, fmt.Sprintf("/usuarios/%d", autor), "")
	if perfil.Code != http.StatusOK || strings.Contains(perfil.Body.String(), `"suspenso"`) {
		t.Fatalf("suspensão não deve ser exposta a outros usuários: %s", perfil.Body)
	}
	if resposta := chamar(t, admin, http.MethodPost, caminho+"/reativar", ""); resposta.Code != http.StatusNoContent {
		t.Fatalf("reativar conta suspensa: %d", resposta.Code)
	}
	conferir(pessoa, 1)
	if resposta := chamar(t, pessoa, http.MethodPost, "/denuncias", corpoPerfil); resposta.Code != http.StatusOK {
		t.Fatalf("nova denúncia após reativação: %d", resposta.Code)
	}
	var historico int
	if erro := db.QueryRow("select count(*) from denuncias where denunciante_id = ? and tipo = 'perfil' and alvo_id = ?", pessoa, autor).Scan(&historico); erro != nil || historico != 3 {
		t.Fatalf("histórico de denúncias: quantidade=%d erro=%v", historico, erro)
	}

	for _, corpo := range []string{
		`{"alvos":[]}`,
		`{"alvos":[{"tipo":"invalido","alvoId":1}]}`,
		`{"alvos":[{"tipo":"perfil","alvoId":0}]}`,
		`{"alvos":[` + strings.Repeat(`{"tipo":"perfil","alvoId":1},`, 100) + `{"tipo":"perfil","alvoId":1}]}`,
	} {
		if resposta := chamar(t, pessoa, http.MethodPost, "/denuncias/consulta", corpo); resposta.Code != http.StatusBadRequest {
			t.Errorf("consulta inválida: status=%d corpo=%s", resposta.Code, resposta.Body)
		}
	}
}
