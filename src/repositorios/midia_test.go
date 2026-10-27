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

	if erro := publicacoes.AdicionarImagem(id, "image/png", []byte("png")); erro != nil {
		t.Fatal(erro)
	}
	if _, dados, erro := publicacoes.BuscarImagem(id, 0); erro != nil || string(dados) != "png" {
		t.Fatalf("imagem: %q %v", dados, erro)
	}
	if erro := publicacoes.Deletar(id); erro != nil {
		t.Fatal(erro)
	}
	if _, erro := midia.Ler(midia.Publicacoes, id); !os.IsNotExist(erro) {
		t.Fatal("excluir a publicação deveria apagar a imagem")
	}
}

func TestVariasImagensPorPublicacao(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "a", false)
	publicacoes := NovoRepositorioDePublicacoes(db)
	id, _ := publicacoes.Criar(modelos.Publicacao{Titulo: "t", Conteudo: "c", AutorID: autor})

	for _, nome := range []string{"um", "dois", "tres", "quatro"} {
		if erro := publicacoes.AdicionarImagem(id, "image/png", []byte(nome)); erro != nil {
			t.Fatal(erro)
		}
	}
	if erro := publicacoes.AdicionarImagem(id, "image/png", []byte("cinco")); !errors.Is(erro, ErrLimiteDeImagens) {
		t.Fatalf("a quinta imagem deveria ser recusada: %v", erro)
	}

	lista, _ := publicacoes.BuscarPorUsuario(autor, autor)
	if len(lista) != 1 || lista[0].TemImagem != true {
		t.Fatalf("publicação sem imagem na listagem: %+v", lista)
	}
	if erro := publicacoes.AnexarRepostagens(lista, autor); erro != nil || lista[0].Imagens != 4 {
		t.Fatalf("total de imagens: %d %v", lista[0].Imagens, erro)
	}

	// remover a primeira promove a segunda
	if erro := publicacoes.RemoverImagem(id, 0); erro != nil {
		t.Fatal(erro)
	}
	for posicao, esperado := range []string{"dois", "tres", "quatro"} {
		if _, dados, erro := publicacoes.BuscarImagem(id, posicao); erro != nil || string(dados) != esperado {
			t.Fatalf("posição %d: %q %v", posicao, dados, erro)
		}
	}
	if _, _, erro := publicacoes.BuscarImagem(id, 3); !errors.Is(erro, sql.ErrNoRows) {
		t.Fatalf("não deveria haver quarta imagem: %v", erro)
	}

	if erro := publicacoes.RemoverImagem(id, 1); erro != nil {
		t.Fatal(erro)
	}
	if _, dados, _ := publicacoes.BuscarImagem(id, 1); string(dados) != "quatro" {
		t.Fatalf("a imagem seguinte deveria subir: %q", dados)
	}

	var extras []uint64
	linhas, _ := db.Query("select id from imagens_extras_publicacoes")
	for linhas.Next() {
		var extra uint64
		linhas.Scan(&extra)
		extras = append(extras, extra)
	}
	linhas.Close()
	if erro := publicacoes.Deletar(id); erro != nil {
		t.Fatal(erro)
	}
	for _, extra := range extras {
		if _, erro := midia.Ler(midia.Extras, extra); !os.IsNotExist(erro) {
			t.Fatal("excluir a publicação deveria apagar as imagens extras")
		}
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

func TestExcluirContaApagaTodasAsMidias(t *testing.T) {
	db := testes.NovoBanco(t)
	autor := testes.CriarUsuario(t, db, "a", false)
	outro := testes.CriarUsuario(t, db, "b", false)

	publicacaoID, _ := NovoRepositorioDePublicacoes(db).Criar(modelos.Publicacao{Titulo: "t", Conteudo: "c", AutorID: autor})
	db.Exec("insert into imagens_publicacoes (publicacao_id, tipo) values (?, 'image/png')", publicacaoID)
	midia.Gravar(midia.Publicacoes, publicacaoID, []byte("p"))

	menor, maior := ordenar(autor, outro)
	conversa, _ := db.Exec("insert into conversas (usuario_menor, usuario_maior, iniciada_por, aceita) values (?, ?, ?, true)", menor, maior, autor)
	conversaID, _ := conversa.LastInsertId()
	mensagem, _ := db.Exec("insert into mensagens (conversa_id, remetente_id, texto) values (?, ?, '')", conversaID, outro)
	mensagemID, _ := mensagem.LastInsertId()
	db.Exec("insert into imagens_mensagens (mensagem_id, tipo) values (?, 'image/png')", mensagemID)
	midia.Gravar(midia.Mensagens, uint64(mensagemID), []byte("m"))

	usuarios := NovoRepositorioDeUsuarios(db)
	usuarios.SalvarFoto(autor, "image/png", []byte("f"))

	if erro := usuarios.Deletar(autor); erro != nil {
		t.Fatal(erro)
	}
	for tipo, id := range map[string]uint64{
		midia.Publicacoes: publicacaoID,
		midia.Mensagens:   uint64(mensagemID),
		midia.Fotos:       autor,
	} {
		if _, erro := midia.Ler(tipo, id); !os.IsNotExist(erro) {
			t.Errorf("o arquivo de %s sobreviveu à exclusão da conta", tipo)
		}
	}
}
