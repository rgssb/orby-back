// Package eventos entrega avisos em tempo real (SSE) aos usuários conectados.
package eventos

import "sync"

// Evento é um aviso curto; o navegador decide o que atualizar ao recebê-lo
type Evento struct {
	Tipo string `json:"tipo"`
	De   uint64 `json:"de,omitempty"`
}

var (
	trava      sync.Mutex
	assinantes = map[uint64]map[chan Evento]struct{}{}
)

// Assinar registra uma conexão do usuário; chame a função devolvida ao encerrar
func Assinar(usuarioID uint64) (<-chan Evento, func()) {
	canal := make(chan Evento, 16)
	trava.Lock()
	if assinantes[usuarioID] == nil {
		assinantes[usuarioID] = map[chan Evento]struct{}{}
	}
	assinantes[usuarioID][canal] = struct{}{}
	trava.Unlock()

	return canal, func() {
		trava.Lock()
		delete(assinantes[usuarioID], canal)
		if len(assinantes[usuarioID]) == 0 {
			delete(assinantes, usuarioID)
		}
		trava.Unlock()
	}
}

// Publicar envia o evento a todas as conexões abertas do usuário, sem nunca bloquear
func Publicar(usuarioID uint64, evento Evento) {
	trava.Lock()
	defer trava.Unlock()
	for canal := range assinantes[usuarioID] {
		select {
		case canal <- evento:
		default:
		}
	}
}
