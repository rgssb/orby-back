package repositorios

import (
	"errors"
	"orby/api/src/testes"
	"testing"
)

func TestBloquearDesfazSeguimentoDosDois(t *testing.T) {
	db := testes.NovoBanco(t)
	a := testes.CriarUsuario(t, db, "a", false)
	b := testes.CriarUsuario(t, db, "b", false)
	usuarios := NovoRepositorioDeUsuarios(db)
	usuarios.Seguir(a, b)
	usuarios.Seguir(b, a)

	bloqueios := NovoRepositorioDeBloqueios(db)
	if erro := bloqueios.Bloquear(a, b); erro != nil {
		t.Fatal(erro)
	}

	var seguidores int
	db.QueryRow("select count(*) from seguidores").Scan(&seguidores)
	if seguidores != 0 {
		t.Fatalf("bloquear deveria desfazer os dois seguimentos, sobraram %d", seguidores)
	}
	bloqueei, bloqueado, _ := bloqueios.Situacao(a, b)
	if !bloqueei || bloqueado {
		t.Fatalf("situação de a: bloqueei=%v bloqueado=%v", bloqueei, bloqueado)
	}
	bloqueei, bloqueado, _ = bloqueios.Situacao(b, a)
	if bloqueei || !bloqueado {
		t.Fatalf("situação de b: bloqueei=%v bloqueado=%v", bloqueei, bloqueado)
	}
}

func TestDirectComBloqueio(t *testing.T) {
	db := testes.NovoBanco(t)
	quemBloqueou := testes.CriarUsuario(t, db, "bloqueador", false)
	bloqueada := testes.CriarUsuario(t, db, "bloqueada", false)
	mensagens := NovoRepositorioDeMensagens(db)

	if erro := NovoRepositorioDeBloqueios(db).Bloquear(quemBloqueou, bloqueada); erro != nil {
		t.Fatal(erro)
	}

	if _, erro := mensagens.Enviar(quemBloqueou, bloqueada, "oi", 0, false); !errors.Is(erro, ErrBloqueado) {
		t.Fatalf("quem bloqueou não pode iniciar conversa: %v", erro)
	}

	for i := 0; i < LimiteAntesDoAceite; i++ {
		if _, erro := mensagens.Enviar(bloqueada, quemBloqueou, "desculpa", 0, false); erro != nil {
			t.Fatalf("pedido de desculpas %d: %v", i+1, erro)
		}
	}
	if _, erro := mensagens.Enviar(bloqueada, quemBloqueou, "desculpa", 0, false); !errors.Is(erro, ErrAguardandoAceite) {
		t.Fatalf("passou do limite sem resposta: %v", erro)
	}

	if _, erro := mensagens.Enviar(quemBloqueou, bloqueada, "tudo bem", 0, false); erro != nil {
		t.Fatalf("quem bloqueou pode responder a um pedido aberto: %v", erro)
	}
}

func TestApagarPedidoDeDesculpasEncerraOPedido(t *testing.T) {
	db := testes.NovoBanco(t)
	quemBloqueou := testes.CriarUsuario(t, db, "bloqueador", false)
	bloqueada := testes.CriarUsuario(t, db, "bloqueada", false)
	mensagens := NovoRepositorioDeMensagens(db)

	NovoRepositorioDeBloqueios(db).Bloquear(quemBloqueou, bloqueada)
	mensagens.Enviar(bloqueada, quemBloqueou, "desculpa", 0, false)

	if erro := mensagens.ExcluirConversa(quemBloqueou, bloqueada); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := mensagens.Enviar(bloqueada, quemBloqueou, "por favor", 0, false); !errors.Is(erro, ErrPedidoEncerrado) {
		t.Fatalf("pedido encerrado não pode ser reaberto: %v", erro)
	}
}

func TestBloqueioMutuoImpedeMensagens(t *testing.T) {
	db := testes.NovoBanco(t)
	a := testes.CriarUsuario(t, db, "a", false)
	b := testes.CriarUsuario(t, db, "b", false)
	bloqueios := NovoRepositorioDeBloqueios(db)
	bloqueios.Bloquear(a, b)
	bloqueios.Bloquear(b, a)

	mensagens := NovoRepositorioDeMensagens(db)
	for _, par := range [][2]uint64{{a, b}, {b, a}} {
		if _, erro := mensagens.Enviar(par[0], par[1], "oi", 0, false); !errors.Is(erro, ErrBloqueado) {
			t.Fatalf("bloqueio mútuo: %v", erro)
		}
	}
}

func TestDesbloquearLiberaConversa(t *testing.T) {
	db := testes.NovoBanco(t)
	a := testes.CriarUsuario(t, db, "a", false)
	b := testes.CriarUsuario(t, db, "b", false)
	bloqueios := NovoRepositorioDeBloqueios(db)
	bloqueios.Bloquear(a, b)
	bloqueios.Desbloquear(a, b)

	if _, erro := NovoRepositorioDeMensagens(db).Enviar(a, b, "oi", 0, false); erro != nil {
		t.Fatalf("após desbloquear: %v", erro)
	}
}

func TestPedidoDeConversaSemSeguimentoMutuo(t *testing.T) {
	db := testes.NovoBanco(t)
	a := testes.CriarUsuario(t, db, "a", false)
	b := testes.CriarUsuario(t, db, "b", false)
	mensagens := NovoRepositorioDeMensagens(db)

	for i := 0; i < LimiteAntesDoAceite; i++ {
		if _, erro := mensagens.Enviar(a, b, "oi", 0, false); erro != nil {
			t.Fatalf("mensagem %d: %v", i+1, erro)
		}
	}
	if _, erro := mensagens.Enviar(a, b, "oi?", 0, false); !errors.Is(erro, ErrAguardandoAceite) {
		t.Fatalf("pedido deveria travar no limite: %v", erro)
	}

	if _, erro := mensagens.Enviar(b, a, "oi!", 0, false); erro != nil {
		t.Fatalf("quem recebeu pode responder: %v", erro)
	}
	if _, erro := mensagens.Enviar(a, b, "que bom", 0, false); erro != nil {
		t.Fatalf("após resposta a conversa segue livre: %v", erro)
	}
}

func TestSeguimentoMutuoDispensaPedido(t *testing.T) {
	db := testes.NovoBanco(t)
	a := testes.CriarUsuario(t, db, "a", false)
	b := testes.CriarUsuario(t, db, "b", false)
	usuarios := NovoRepositorioDeUsuarios(db)
	usuarios.Seguir(a, b)
	usuarios.Seguir(b, a)
	mensagens := NovoRepositorioDeMensagens(db)

	for i := 0; i < LimiteAntesDoAceite+2; i++ {
		if _, erro := mensagens.Enviar(a, b, "oi", 0, false); erro != nil {
			t.Fatalf("mensagem %d: %v", i+1, erro)
		}
	}
}

func TestMensagemInvalida(t *testing.T) {
	db := testes.NovoBanco(t)
	a := testes.CriarUsuario(t, db, "a", false)
	mensagens := NovoRepositorioDeMensagens(db)

	if _, erro := mensagens.Enviar(a, a, "eco", 0, false); !errors.Is(erro, ErrMensagemParaSiMesmo) {
		t.Fatalf("para si mesmo: %v", erro)
	}
	if _, erro := mensagens.Enviar(a, 9999, "alô", 0, false); !errors.Is(erro, ErrUsuarioInexistente) {
		t.Fatalf("usuário inexistente: %v", erro)
	}
}
