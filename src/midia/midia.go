// Package midia guarda fotos, banners e imagens em disco; o banco só registra o tipo do arquivo.
package midia

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Tipos de mídia; cada um vira uma pasta dentro de Dir
const (
	Fotos       = "fotos"
	Banners     = "banners"
	Publicacoes = "publicacoes"
	Mensagens   = "mensagens"
	Extras      = "extras"
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

// Antigos lista os ids dos arquivos do tipo cuja última modificação é anterior a limite.
// Nomes que não são números (como os temporários de Gravar) são ignorados.
func Antigos(tipo string, limite time.Time) ([]uint64, error) {
	entradas, erro := os.ReadDir(filepath.Join(Dir, tipo))
	if os.IsNotExist(erro) {
		return nil, nil
	}
	if erro != nil {
		return nil, erro
	}

	var ids []uint64
	for _, entrada := range entradas {
		id, erro := strconv.ParseUint(entrada.Name(), 10, 64)
		if erro != nil || entrada.IsDir() {
			continue
		}
		if info, erro := entrada.Info(); erro == nil && info.ModTime().Before(limite) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
