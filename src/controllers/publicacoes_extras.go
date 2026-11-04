package controllers

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/url"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
)

// LimiteImagemPublicacao é o tamanho máximo da imagem anexada a uma publicação
const LimiteImagemPublicacao = 4 << 20

var expressaoDeHashtag = regexp.MustCompile(`^[\p{L}\p{N}_]{1,50}$`)

// PesquisarPublicacoes procura publicações por texto ou, quando a busca começa com #, por hashtag
func PesquisarPublicacoes(w http.ResponseWriter, r *http.Request) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}

	termo := strings.TrimSpace(r.URL.Query().Get("q"))
	autor := strings.TrimPrefix(strings.TrimSpace(r.URL.Query().Get("autor")), "@")
	inicio, fim := r.URL.Query().Get("inicio"), r.URL.Query().Get("fim")
	if utf8.RuneCountInString(autor) > 50 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("O nick deve ter no máximo 50 caracteres"))
		return
	}
	for _, data := range []string{inicio, fim} {
		if data != "" {
			if _, erro := time.Parse("2006-01-02", data); erro != nil {
				respostas.Erro(w, http.StatusBadRequest, errors.New("Data inválida"))
				return
			}
		}
	}
	if inicio != "" && fim != "" && inicio > fim {
		respostas.Erro(w, http.StatusBadRequest, errors.New("A data inicial deve ser anterior à final"))
		return
	}
	hashtag := strings.HasPrefix(termo, "#")
	if hashtag {
		termo = strings.TrimPrefix(termo, "#")
		if !expressaoDeHashtag.MatchString(termo) {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Hashtag inválida"))
			return
		}
	} else if (termo != "" && utf8.RuneCountInString(termo) < 2) || utf8.RuneCountInString(termo) > 100 || (termo == "" && autor == "" && inicio == "" && fim == "") {
		respostas.Erro(w, http.StatusBadRequest, errors.New("A busca deve ter entre 2 e 100 caracteres"))
		return
	}

	limite, _ := strconv.ParseUint(r.URL.Query().Get("limite"), 10, 64)
	if limite == 0 || limite > 50 {
		limite = 20
	}
	deslocamento, _ := strconv.ParseUint(r.URL.Query().Get("deslocamento"), 10, 64)

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	publicacoes, erro := repositorio.PesquisarComFiltros(usuarioID, termo, hashtag, r.URL.Query().Get("ordem"), autor, inicio, fim, limite, deslocamento)
	if erro == nil {
		erro = repositorio.AnexarRepostagens(publicacoes, usuarioID)
	}
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusOK, publicacoes)
}

// posicaoDaImagem lê a posição da imagem na URL; sem ela, é a primeira
func posicaoDaImagem(r *http.Request) (int, error) {
	texto, existe := mux.Vars(r)["posicao"]
	if !existe {
		return 0, nil
	}
	return strconv.Atoi(texto)
}

// BuscarImagemDaPublicacao devolve uma das imagens anexadas a uma publicação
func BuscarImagemDaPublicacao(w http.ResponseWriter, r *http.Request) {
	publicacaoID, erro := strconv.ParseUint(mux.Vars(r)["publicacaoId"], 10, 64)
	posicao, erroPosicao := posicaoDaImagem(r)
	if erro != nil || erroPosicao != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Endereço inválido"))
		return
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()

	visitanteID, _ := autenticacao.ExtrairUsuarioID(r)
	if pode, _ := verificarLeituraPublicacao(w, db, publicacaoID, visitanteID, "imagem"); !pode {
		return
	}

	tipo, dados, erro := repositorios.NovoRepositorioDePublicacoes(db).BuscarImagem(publicacaoID, posicao)
	if erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Imagem não encontrada"))
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	w.Header().Set("Content-Type", tipo)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	w.Write(dados)
}

// AdicionarImagemDaPublicacao anexa uma imagem (corpo bruto) a uma publicação do usuário autenticado, até 4
func AdicionarImagemDaPublicacao(w http.ResponseWriter, r *http.Request) {
	db, repositorio, publicacaoID, ok := publicacaoDoUsuarioAutenticado(w, r)
	if !ok {
		return
	}
	defer db.Close()

	dados, erro := io.ReadAll(http.MaxBytesReader(w, r.Body, LimiteImagemPublicacao))
	if erro != nil {
		respostas.Erro(w, http.StatusRequestEntityTooLarge, errors.New("A imagem deve ter no máximo 4 MB"))
		return
	}

	if len(dados) == 0 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Nenhuma imagem foi enviada"))
		return
	}

	tipo := http.DetectContentType(dados)
	if !tiposDeFotoPermitidos[tipo] {
		respostas.Erro(w, http.StatusUnsupportedMediaType, errors.New("Envie uma imagem JPG, PNG, WEBP ou GIF"))
		return
	}

	descricao, erro := url.PathUnescape(r.Header.Get("X-Orby-Descricao"))
	if erro != nil || !utf8.ValidString(descricao) || utf8.RuneCountInString(descricao) > 500 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("A descrição da imagem deve ter no máximo 500 caracteres"))
		return
	}
	descricao = strings.TrimSpace(descricao)
	if erro = repositorio.AdicionarImagemComDescricao(publicacaoID, tipo, dados, descricao); erro != nil {
		if errors.Is(erro, repositorios.ErrLimiteDeImagens) {
			respostas.Erro(w, http.StatusConflict, erro)
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// RemoverImagemDaPublicacao apaga uma imagem de uma publicação do usuário autenticado
func RemoverImagemDaPublicacao(w http.ResponseWriter, r *http.Request) {
	posicao, erro := posicaoDaImagem(r)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Endereço inválido"))
		return
	}

	db, repositorio, publicacaoID, ok := publicacaoDoUsuarioAutenticado(w, r)
	if !ok {
		return
	}
	defer db.Close()

	if erro = repositorio.RemoverImagem(publicacaoID, posicao); erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Imagem não encontrada"))
			return
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}

	respostas.JSON(w, http.StatusNoContent, nil)
}

// publicacaoDoUsuarioAutenticado confere que a publicação da URL pertence a quem fez a requisição.
// Em caso de sucesso, quem chama deve fechar o db devolvido.
func publicacaoDoUsuarioAutenticado(w http.ResponseWriter, r *http.Request) (*sql.DB, *repositorios.Publicacoes, uint64, bool) {
	usuarioID, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return nil, nil, 0, false
	}

	publicacaoID, erro := strconv.ParseUint(mux.Vars(r)["publicacaoId"], 10, 64)
	if erro != nil {
		respostas.Erro(w, http.StatusBadRequest, erro)
		return nil, nil, 0, false
	}

	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return nil, nil, 0, false
	}

	repositorio := repositorios.NovoRepositorioDePublicacoes(db)
	publicacao, erro := repositorio.BuscarPorID(publicacaoID)
	if erro != nil {
		db.Close()
		if errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusNotFound, errors.New("Publicação não encontrada"))
			return nil, nil, 0, false
		}
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return nil, nil, 0, false
	}

	if publicacao.AutorID != usuarioID {
		db.Close()
		respostas.Erro(w, http.StatusForbidden, errors.New("Não é possível alterar algo que não é seu"))
		return nil, nil, 0, false
	}

	return db, repositorio, publicacaoID, true
}

// verificarAcessoPublicacao responde 404 (sem revelar a existência) quando o usuário não pode ver a publicação
func verificarAcessoPublicacao(w http.ResponseWriter, db *sql.DB, publicacaoID, usuarioID uint64) bool {
	pode, erro := repositorios.NovoRepositorioDePublicacoes(db).PodeVer(publicacaoID, usuarioID)
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return false
	}

	if !pode {
		respostas.Erro(w, http.StatusNotFound, errors.New("Publicação não encontrada"))
		return false
	}
	return true
}
