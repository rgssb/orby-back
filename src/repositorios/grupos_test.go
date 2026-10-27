package repositorios

import (
	"errors"
	"orby/api/src/eventos"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"testing"
	"time"
)

func TestGruposCriarEntrarEConversar(t *testing.T) {
	db := testes.NovoBanco(t)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	caio := testes.CriarUsuario(t, db, "caio", false)
	repo := NovoRepositorioDeGrupos(db)

	if _, erro := repo.Criar(ana, "vazio", nil); !errors.Is(erro, ErrGrupoSemMembros) {
		t.Fatalf("grupo sem ninguém deveria falhar: %v", erro)
	}
	db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", caio, ana)
	if _, erro := repo.Criar(ana, "g", []uint64{beto, caio}); !errors.Is(erro, ErrBloqueado) {
		t.Fatalf("bloqueio deveria impedir: %v", erro)
	}
	db.Exec("delete from bloqueios")

	doBeto, sair := eventos.Assinar(beto)
	defer sair()
	id, erro := repo.Criar(ana, "Amigos", []uint64{beto, beto, ana})
	if erro != nil {
		t.Fatal(erro)
	}
	select {
	case e := <-doBeto:
		if e.Tipo != "grupo" || e.De != id {
			t.Fatalf("evento inesperado: %+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("beto deveria ser avisado")
	}

	if _, erro = repo.Enviar(caio, id, "oi"); !errors.Is(erro, ErrGrupoInexistente) {
		t.Fatalf("não membro não envia: %v", erro)
	}
	if _, erro = repo.Enviar(ana, id, "olá, pessoal"); erro != nil {
		t.Fatal(erro)
	}
	lista, _ := repo.Listar(beto)
	if len(lista) != 1 || lista[0].NaoLidas != 1 || lista[0].Membros != 2 {
		t.Fatalf("lista inesperada: %+v", lista)
	}
	dialogo, erro := repo.Abrir(beto, id, 0)
	if erro != nil || len(dialogo.MensagensDoGrupo) != 1 || dialogo.EuPossoAdicionar {
		t.Fatalf("dialogo inesperado: %+v %v", dialogo, erro)
	}
	if lista, _ = repo.Listar(beto); lista[0].NaoLidas != 0 {
		t.Fatal("abrir deveria marcar como lido")
	}
}

func TestGruposPermissoesLimiteESaida(t *testing.T) {
	db := testes.NovoBanco(t)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	caio := testes.CriarUsuario(t, db, "caio", false)
	duda := testes.CriarUsuario(t, db, "duda", false)
	repo := NovoRepositorioDeGrupos(db)
	id, _ := repo.Criar(ana, "Amigos", []uint64{beto})

	if erro := repo.Adicionar(beto, id, caio); !errors.Is(erro, ErrSemPermissao) {
		t.Fatalf("membro comum não adiciona: %v", erro)
	}
	if erro := repo.DefinirPermissao(beto, id, beto, true); !errors.Is(erro, ErrSemPermissao) {
		t.Fatalf("só o criador concede: %v", erro)
	}
	if erro := repo.DefinirPermissao(ana, id, beto, true); erro != nil {
		t.Fatal(erro)
	}
	db.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", caio, beto)
	if erro := repo.Adicionar(beto, id, caio); !errors.Is(erro, ErrBloqueado) {
		t.Fatalf("bloqueio entre quem adiciona e o novo: %v", erro)
	}
	if erro := repo.Adicionar(beto, id, duda); erro != nil {
		t.Fatal(erro)
	}
	if erro := repo.Adicionar(beto, id, duda); !errors.Is(erro, ErrJaEMembro) {
		t.Fatalf("já é membro: %v", erro)
	}
	if erro := repo.Remover(beto, id, duda); !errors.Is(erro, ErrSemPermissao) {
		t.Fatalf("só o criador remove outros: %v", erro)
	}
	if erro := repo.Remover(ana, id, duda); erro != nil {
		t.Fatal(erro)
	}

	// o criador sai e o posto passa ao membro mais antigo
	if erro := repo.Remover(ana, id, ana); erro != nil {
		t.Fatal(erro)
	}
	var criador uint64
	db.QueryRow("select criador_id from grupos where id = ?", id).Scan(&criador)
	if criador != beto {
		t.Fatalf("criador deveria ser beto, é %d", criador)
	}
	if erro := repo.Remover(beto, id, beto); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := repo.Abrir(beto, id, 0); !errors.Is(erro, ErrGrupoInexistente) {
		t.Fatalf("grupo vazio deveria sumir: %v", erro)
	}
}

func TestGruposLimiteDeMembros(t *testing.T) {
	db := testes.NovoBanco(t)
	ana := testes.CriarUsuario(t, db, "ana", false)
	repo := NovoRepositorioDeGrupos(db)
	var ids []uint64
	for i := 0; i < modelos.MaximoDeMembros-1; i++ {
		ids = append(ids, testes.CriarUsuario(t, db, "u"+string(rune('a'+i/26))+string(rune('a'+i%26)), false))
	}
	id, erro := repo.Criar(ana, "Cheio", ids)
	if erro != nil {
		t.Fatal(erro)
	}
	extra := testes.CriarUsuario(t, db, "extra", false)
	if erro = repo.Adicionar(ana, id, extra); !errors.Is(erro, ErrGrupoCheio) {
		t.Fatalf("deveria estar cheio: %v", erro)
	}
	if _, erro = repo.Criar(ana, "Maior", append(ids, extra)); !errors.Is(erro, ErrGrupoCheio) {
		t.Fatalf("criar acima do limite: %v", erro)
	}
}

func TestGruposContaDoCriadorExcluida(t *testing.T) {
	db := testes.NovoBanco(t)
	ana := testes.CriarUsuario(t, db, "ana", false)
	beto := testes.CriarUsuario(t, db, "beto", false)
	repo := NovoRepositorioDeGrupos(db)
	comBeto, _ := repo.Criar(ana, "Com Beto", []uint64{beto})
	sozinho, _ := repo.Criar(beto, "Do Beto", []uint64{ana})

	if erro := NovoRepositorioDeUsuarios(db).Deletar(ana); erro != nil {
		t.Fatal(erro)
	}
	var criador uint64
	if erro := db.QueryRow("select criador_id from grupos where id = ?", comBeto).Scan(&criador); erro != nil || criador != beto {
		t.Fatalf("o grupo deveria passar ao beto: %d %v", criador, erro)
	}
	if _, erro := repo.Abrir(beto, sozinho, 0); erro != nil {
		t.Fatalf("grupo de outro criador deveria continuar: %v", erro)
	}
}
