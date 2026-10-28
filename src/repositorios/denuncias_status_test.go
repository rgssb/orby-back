package repositorios

import (
	"errors"
	"fmt"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"sync"
	"testing"
)

func TestDenunciasConcorrentesNaoDuplicam(t *testing.T) {
	db := testes.NovoBanco(t)
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	pessoa := testes.CriarUsuario(t, db, "pessoa", false)
	repo := NovoRepositorioDeDenuncias(db)
	inicio := make(chan struct{})
	resultados := make(chan error, 8)
	var grupo sync.WaitGroup
	for i := 0; i < cap(resultados); i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			resultados <- repo.Registrar(pessoa, modelos.DenunciaPerfil, alvo, "spam")
		}()
	}
	close(inicio)
	grupo.Wait()
	close(resultados)
	sucessos, duplicadas := 0, 0
	for erro := range resultados {
		switch {
		case erro == nil:
			sucessos++
		case errors.Is(erro, ErrJaDenunciado):
			duplicadas++
		default:
			t.Fatalf("erro inesperado: %v", erro)
		}
	}
	if sucessos != 1 || duplicadas != 7 {
		t.Fatalf("sucessos=%d duplicadas=%d", sucessos, duplicadas)
	}
}

func TestFalhaAoNotificarDesfazDenuncia(t *testing.T) {
	db := testes.NovoBanco(t)
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	repo := NovoRepositorioDeDenuncias(db)
	for i := 0; i < 9; i++ {
		pessoa := testes.CriarUsuario(t, db, fmt.Sprintf("pessoa%d", i), false)
		if erro := repo.Registrar(pessoa, modelos.DenunciaPerfil, alvo, "spam"); erro != nil {
			t.Fatal(erro)
		}
	}
	decimo := testes.CriarUsuario(t, db, "decimo", false)
	if _, erro := db.Exec("drop table notificacoes"); erro != nil {
		t.Fatal(erro)
	}
	if erro := repo.Registrar(decimo, modelos.DenunciaPerfil, alvo, "spam"); erro == nil {
		t.Fatal("falha ao criar alerta deveria ser informada")
	}
	var quantidade int
	if erro := db.QueryRow("select count(*) from denuncias where denunciante_id = ?", decimo).Scan(&quantidade); erro != nil || quantidade != 0 {
		t.Fatalf("denúncia não foi desfeita: quantidade=%d erro=%v", quantidade, erro)
	}
}
