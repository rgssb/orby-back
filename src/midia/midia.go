// Package midia guarda fotos, banners e imagens em disco; o banco só registra o tipo do arquivo.
package midia

import (
	"os"
	"path/filepath"
	"strconv"
)

// Tipos de mídia; cada um vira uma pasta dentro de Dir
const (
	Fotos       = "fotos"
	Banners     = "banners"
	Publicacoes = "publicacoes"
	Mensagens   = "mensagens"
)

// Dir é a pasta raiz das mídias (variável MEDIA_DIR)
var Dir = "midia"

func caminho(tipo string, id uint64) string {
	return filepath.Join(Dir, tipo, strconv.FormatUint(id, 10))
}

// Gravar salva (ou troca) o arquivo de forma atômica
func Gravar(tipo string, id uint64, dados []byte) error {
	destino := caminho(tipo, id)
	if erro := os.MkdirAll(filepath.Dir(destino), 0o755); erro != nil {
		return erro
	}

	temporario, erro := os.CreateTemp(filepath.Dir(destino), ".envio-*")
	if erro != nil {
		return erro
	}
	defer os.Remove(temporario.Name())

	if _, erro = temporario.Write(dados); erro != nil {
		temporario.Close()
		return erro
	}
	if erro = temporario.Close(); erro != nil {
		return erro
	}
	return os.Rename(temporario.Name(), destino)
}

// Ler devolve o conteúdo do arquivo; os.ErrNotExist quando ele não existe
func Ler(tipo string, id uint64) ([]byte, error) {
	return os.ReadFile(caminho(tipo, id))
}

// Remover apaga o arquivo; não é erro ele já não existir
func Remover(tipo string, id uint64) error {
	if erro := os.Remove(caminho(tipo, id)); erro != nil && !os.IsNotExist(erro) {
		return erro
	}
	return nil
}
