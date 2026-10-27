package repositorios

import (
	"bytes"
	"database/sql"
	"errors"
	"orby/api/src/banco"
	"orby/api/src/config"
	"orby/api/src/midia"
	"orby/api/src/modelos"
	"orby/api/src/testes"
	"os"
	"testing"
)

func TestFotoEBannerFicamEmDisco(t *testing.T) {
	db := testes.NovoBanco(t)
	id := testes.CriarUsuario(t, db, "a", false)
	usuarios := NovoRepositorioDeUsuarios(db)

	if _, _, erro := usuarios.BuscarFoto(id); !errors.Is(erro, sql.ErrNoRows) {
		t.Fatalf("sem foto: %v", erro)
	}

	if erro := usuarios.SalvarFoto(id, "image/png", []byte("png")); erro != nil {
		t.Fatal(erro)
	}
	tipo, dados, erro := usuarios.BuscarFoto(id)
	if erro != nil || tipo != "image/png" || !bytes.Equal(dados, []byte("png")) {
		t.Fatalf("foto: %q %q %v", tipo, dados, erro)
	}

	usuarios.SalvarBannerImagem(id, "image/jpeg", []byte("jpg"))
	if _, erro = midia.Ler(midia.Banners, id); erro != nil {
		t.Fatalf("banner deveria estar em disco: %v", erro)
	}
	if erro = usuarios.SalvarBannerPreset(id, "aurora"); erro != nil {
		t.Fatal(erro)
	}
	if _, erro = midia.Ler(midia.Banners, id); !os.IsNotExist(erro) {
		t.Fatal("trocar por tema pronto deveria apagar a imagem do banner")
	}
	if _, _, erro = usuarios.BuscarBannerImagem(id); !errors.Is(erro, sql.ErrNoRows) {
		t.Fatalf("sem imagem de banner: %v", erro)
	}

	if erro = usuarios.RemoverFoto(id); erro != nil {
		t.Fatal(erro)
	}
	if _, erro = midia.Ler(midia.Fotos, id); !os.IsNotExist(erro) {
		t.Fatal("remover a foto deveria apagar o arquivo")
	}
}

func TestExcluirPublicacaoApagaImagem(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "a", false)
	publicacoes := NovoRepositorioDePublicacoes(db)
	id, _ := publicacoes.Criar(modelos.Publicacao{Titulo: "t", Conteudo: "c", AutorID: autor})

	if erro := publicacoes.SalvarImagem(id, "image/png", []byte("png")); erro != nil {
		t.Fatal(erro)
	}
	if _, dados, erro := publicacoes.BuscarImagem(id); erro != nil || string(dados) != "png" {
		t.Fatalf("imagem: %q %v", dados, erro)
	}
	if erro := publicacoes.Deletar(id); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := midia.Ler(midia.Publicacoes, id); !os.IsNotExist(erro) {
		t.Fatal("excluir a publicação deveria apagar a imagem")
	}
}

func TestMigrarMidiaMoveBlobsParaDisco(t *testing.T) {
	db := testes.NovoBanco(t)
	id := testes.CriarUsuario(t, db, "a", false)

	// Esquema antigo: foto guardada no banco
	if _, erro := db.Exec("alter table fotos_usuarios add column dados mediumblob null"); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := db.Exec("insert into fotos_usuarios (usuario_id, tipo, dados) values (?, 'image/png', 'antiga')", id); erro != nil {
		t.Fatal(erro)
	}

	config.StringConexaoBanco = os.Getenv("ORBY_TEST_DSN")
	if erro := banco.Migrar(); erro != nil {
		t.Fatal(erro)
	}

	_, dados, erro := NovoRepositorioDeUsuarios(db).BuscarFoto(id)
	if erro != nil || string(dados) != "antiga" {
		t.Fatalf("foto migrada: %q %v", dados, erro)
	}
	var existe int
	db.QueryRow("select count(*) from information_schema.columns where table_schema = database() and table_name = 'fotos_usuarios' and column_name = 'dados'").Scan(&existe)
	if existe != 0 {
		t.Fatal("a coluna dados deveria ter sido removida")
	}
}
