// Package suspensoes guarda em memória quais contas estão suspensas, para barrar as requisições sem consultar o banco.
package suspensoes

import (
	"database/sql"
	"sync"
)

var (
	trava sync.RWMutex
	ids   = map[uint64]bool{}
)

// Carregar lê do banco as contas suspensas
func Carregar(db *sql.DB) error {
	linhas, erro := db.Query("select id from usuarios where suspenso_em is not null")
	if erro != nil {
		return erro
	}
	defer linhas.Close()

	novas := map[uint64]bool{}
	for linhas.Next() {
		var id uint64
		if erro = linhas.Scan(&id); erro != nil {
			return erro
		}
		novas[id] = true
	}
	if erro = linhas.Err(); erro != nil {
		return erro
	}
	trava.Lock()
	ids = novas
	trava.Unlock()
	return nil
}

// Definir marca ou desmarca a conta como suspensa
func Definir(id uint64, suspensa bool) {
	trava.Lock()
	defer trava.Unlock()
	if suspensa {
		ids[id] = true
	} else {
		delete(ids, id)
	}
}

// Suspensa diz se a conta está suspensa
func Suspensa(id uint64) bool {
	trava.RLock()
	defer trava.RUnlock()
	return ids[id]
}
