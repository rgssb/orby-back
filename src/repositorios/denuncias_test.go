package repositorios

import (
	"errors"
	"fmt"
	"orby/api/src/modelos"
	"orby/api/src/suspensoes"
	"orby/api/src/testes"
	"testing"
)

func TestDenunciasAlertaRevisaoEResolucao(t *testing.T) {
	db := testes.NovoBanco(t)
	alvo := testes.CriarUsuario(t, db, "alvo", false)
	adm := testes.CriarUsuario(t, db, "adm", false)
	db.Exec("update usuarios set admin = 1 where id = ?", adm)
	repo := NovoRepositorioDeDenuncias(db)

	if erro := repo.Registrar(alvo, modelos.DenunciaPerfil, alvo, "spam"); !errors.Is(erro, ErrDenunciaPropria) {
		t.Fatalf("denunciar a si mesmo deveria falhar: %v", erro)
	}
	if erro := repo.Registrar(adm, modelos.DenunciaPerfil, alvo, "xyz"); !errors.Is(erro, ErrDenunciaInvalida) {
		t.Fatalf("motivo inválido deveria falhar: %v", erro)
	}
	if erro := repo.Registrar(adm, modelos.DenunciaPublicacao, 999999, "spam"); !errors.Is(erro, ErrDenunciaNaoAchada) {
		t.Fatalf("conteúdo inexistente deveria falhar: %v", erro)
	}

	avisos := func(tipo string) (n int) {
		db.QueryRow("select count(*) from notificacoes where usuario_id = ? and ator_id = ? and tipo = ?", adm, alvo, tipo).Scan(&n)
		return
	}
	denunciar := func(quantas int, desde int) {
		for i := desde; i < desde+quantas; i++ {
			id := testes.CriarUsuario(t, db, fmt.Sprintf("d%d", i), false)
			if erro := repo.Registrar(id, modelos.DenunciaPerfil, alvo, "spam"); erro != nil {
				t.Fatal(erro)
			}
		}
	}

	denunciar(9, 0)
	if avisos(modelos.NotificacaoPerfilEmAlerta) != 0 {
		t.Fatal("com 9 denúncias ainda não deveria haver alerta")
	}
	denunciar(1, 9)
	if avisos(modelos.NotificacaoPerfilEmAlerta) != 1 || avisos(modelos.NotificacaoPerfilEmRevisao) != 0 {
		t.Fatal("com 10 denúncias deveria entrar em alerta")
	}
	if perfis, erro := repo.ListarPerfis(); erro != nil || len(perfis) != 1 || perfis[0].Nivel != "alerta" {
		t.Fatalf("lista inesperada: %+v %v", perfis, erro)
	}
	denunciar(10, 10)
	if avisos(modelos.NotificacaoPerfilEmRevisao) != 1 || avisos(modelos.NotificacaoPerfilEmAlerta) != 1 {
		t.Fatal("com 20 denúncias deveria ir para revisão sem repetir o alerta")
	}
	if perfis, _ := repo.ListarPerfis(); perfis[0].Nivel != "revisao" || perfis[0].Total != 20 {
		t.Fatalf("nível inesperado: %+v", perfis)
	}

	// a mesma pessoa denunciar de novo o mesmo perfil não conta duas vezes
	var primeiro uint64
	db.QueryRow("select denunciante_id from denuncias limit 1").Scan(&primeiro)
	repo.Registrar(primeiro, modelos.DenunciaPerfil, alvo, "assedio")
	if perfis, _ := repo.ListarPerfis(); perfis[0].Total != 20 {
		t.Fatalf("denúncia repetida não pode contar: %+v", perfis)
	}

	if erro := repo.Resolver(alvo); erro != nil {
		t.Fatal(erro)
	}
	if perfis, _ := repo.ListarPerfis(); len(perfis) != 0 || avisos(modelos.NotificacaoPerfilEmRevisao) != 0 {
		t.Fatal("resolver deveria limpar a lista e os avisos")
	}
	if ehAdmin, _ := EhAdmin(db, alvo); ehAdmin {
		t.Fatal("alvo não é admin")
	}
	if ehAdmin, _ := EhAdmin(db, adm); !ehAdmin {
		t.Fatal("adm deveria ser admin")
	}
}

func TestAdministracaoSuspender(t *testing.T) {
	db := testes.NovoBanco(t)
	ana := testes.CriarUsuario(t, db, "ana", false)
	adm := testes.CriarUsuario(t, db, "adm", false)
	db.Exec("update usuarios set admin = 1 where id = ?", adm)
	repo := NovoRepositorioDeAdministracao(db)

	if erro := repo.Suspender(adm); !errors.Is(erro, ErrNaoSuspendeAdmin) {
		t.Fatalf("admin não pode ser suspenso: %v", erro)
	}
	if erro := repo.Suspender(999999); !errors.Is(erro, ErrUsuarioNaoEncontrado) {
		t.Fatalf("usuário inexistente: %v", erro)
	}
	if erro := repo.Suspender(ana); erro != nil {
		t.Fatal(erro)
	}
	if !suspensoes.Suspensa(ana) {
		t.Fatal("conta deveria constar como suspensa")
	}
	if contas, _ := repo.ListarSuspensos(); len(contas) != 1 || contas[0].UsuarioID != ana {
		t.Fatalf("lista inesperada: %+v", contas)
	}
	if e, erro := repo.Estatisticas(); erro != nil || e.Suspensos != 1 {
		t.Fatalf("estatísticas inesperadas: %+v %v", e, erro)
	}
	if erro := repo.Reativar(ana); erro != nil || suspensoes.Suspensa(ana) {
		t.Fatalf("reativar falhou: %v", erro)
	}
}
