package midia

import (
	"bytes"
	"os"
	"testing"
)

func TestGravarLerRemover(t *testing.T) {
	Dir = t.TempDir()

	if _, erro := Ler(Fotos, 1); !os.IsNotExist(erro) {
		t.Fatalf("arquivo inexistente deveria devolver ErrNotExist: %v", erro)
	}

	if erro := Gravar(Fotos, 1, []byte("um")); erro != nil {
		t.Fatal(erro)
	}
	if erro := Gravar(Fotos, 1, []byte("dois")); erro != nil {
		t.Fatal(erro)
	}
	if dados, erro := Ler(Fotos, 1); erro != nil || !bytes.Equal(dados, []byte("dois")) {
		t.Fatalf("dados = %q, erro = %v", dados, erro)
	}
	if _, erro := Ler(Banners, 1); !os.IsNotExist(erro) {
		t.Fatal("tipos diferentes não compartilham arquivos")
	}

	if erro := Remover(Fotos, 1); erro != nil {
		t.Fatal(erro)
	}
	if erro := Remover(Fotos, 1); erro != nil {
		t.Fatalf("remover duas vezes não deveria falhar: %v", erro)
	}
}
