package banco

import (
	"orby/api/src/midia"
	"orby/api/src/testes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestLimparMidiaOrfaPreservaRegistradosERecentes(t *testing.T) {
	db := testes.NovoBanco(t)
	usuario := testes.CriarUsuario(t, db, "autor", false)
	db.Exec("insert into fotos_usuarios (usuario_id, tipo) values (?, 'image/png')", usuario)

	envelhecer := func(tipo string, id uint64) {
		antigo := time.Now().Add(-2 * time.Hour)
		caminho := filepath.Join(midia.Dir, tipo, strconv.FormatUint(id, 10))
		if erro := os.Chtimes(caminho, antigo, antigo); erro != nil {
			t.Fatal(erro)
		}
	}

	midia.Gravar(midia.Fotos, usuario, []byte("foto"))
	midia.Gravar(midia.Mensagens, 999, []byte("orfa antiga"))
	midia.Gravar(midia.Mensagens, 1000, []byte("orfa recente"))
	envelhecer(midia.Fotos, usuario)
	envelhecer(midia.Mensagens, 999)

	removidos, erro := LimparMidiaOrfa(db)
	if erro != nil || removidos != 1 {
		t.Fatalf("removidos = %d, erro = %v", removidos, erro)
	}
	if _, erro = midia.Ler(midia.Mensagens, 999); !os.IsNotExist(erro) {
		t.Error("a órfã antiga deveria ter sido apagada")
	}
	if _, erro = midia.Ler(midia.Mensagens, 1000); erro != nil {
		t.Error("a órfã recente deveria ser preservada")
	}
	if _, erro = midia.Ler(midia.Fotos, usuario); erro != nil {
		t.Error("a foto registrada deveria ser preservada")
	}
}
