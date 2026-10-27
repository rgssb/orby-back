package banco

import (
	"database/sql"
	"fmt"
	"orby/api/src/midia"
	"time"
)

// idadeMinimaDoOrfao evita apagar um arquivo recém-gravado cujo registro ainda não foi confirmado no banco
const idadeMinimaDoOrfao = time.Hour

// LimparMidiaOrfa apaga os arquivos de mídia sem registro no banco (por exemplo, imagens de mensagens
// removidas por cascata quando um usuário ou conversa é excluído) e devolve quantos foram removidos.
func LimparMidiaOrfa(db *sql.DB) (int, error) {
	removidos := 0
	for _, t := range tabelasDeMidia {
		candidatos, erro := midia.Antigos(t.tipo, time.Now().Add(-idadeMinimaDoOrfao))
		if erro != nil {
			return removidos, fmt.Errorf("limpeza de mídia (%s): %w", t.tipo, erro)
		}
		if len(candidatos) == 0 {
			continue
		}

		existentes := map[uint64]bool{}
		linhas, erro := db.Query(fmt.Sprintf("select %s from %s", t.chave, t.tabela))
		if erro != nil {
			return removidos, fmt.Errorf("limpeza de mídia (%s): %w", t.tipo, erro)
		}
		for linhas.Next() {
			var id uint64
			if erro = linhas.Scan(&id); erro == nil {
				existentes[id] = true
			}
		}
		linhas.Close()

		for _, id := range candidatos {
			if existentes[id] {
				continue
			}
			if erro = midia.Remover(t.tipo, id); erro != nil {
				return removidos, erro
			}
			removidos++
		}
	}
	return removidos, nil
}
