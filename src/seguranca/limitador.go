package seguranca

import (
	"sync"
	"time"
)

// Limitador conta tentativas por chave dentro de uma janela de tempo (em memória)
type Limitador struct {
	mu         sync.Mutex
	maximo     int
	janela     time.Duration
	tentativas map[string][]time.Time
}

// NovoLimitador cria um limitador que aceita `maximo` tentativas a cada `janela`
func NovoLimitador(maximo int, janela time.Duration) *Limitador {
	return &Limitador{
		maximo:     maximo,
		janela:     janela,
		tentativas: make(map[string][]time.Time),
	}
}

func (l *Limitador) recentes(chave string) []time.Time {
	limite := time.Now().Add(-l.janela)
	recentes := l.tentativas[chave][:0]
	for _, instante := range l.tentativas[chave] {
		if instante.After(limite) {
			recentes = append(recentes, instante)
		}
	}

	if len(recentes) == 0 {
		delete(l.tentativas, chave)
		return nil
	}

	l.tentativas[chave] = recentes
	return recentes
}

// Bloqueado informa se a chave já atingiu o máximo de tentativas
func (l *Limitador) Bloqueado(chave string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recentes(chave)) >= l.maximo
}

// Registrar anota uma tentativa para a chave
func (l *Limitador) Registrar(chave string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.recentes(chave)
	l.tentativas[chave] = append(l.tentativas[chave], time.Now())
}

// Reiniciar zera as tentativas da chave
func (l *Limitador) Reiniciar(chave string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.tentativas, chave)
}
