package seguranca

import (
	"testing"
	"time"
)

func TestLimitadorBloqueiaAposMaximo(t *testing.T) {
	l := NovoLimitador(2, time.Minute)
	l.Registrar("ip")
	if l.Bloqueado("ip") {
		t.Fatal("não deveria bloquear antes do máximo")
	}
	l.Registrar("ip")
	if !l.Bloqueado("ip") {
		t.Fatal("deveria bloquear ao atingir o máximo")
	}
	if l.Bloqueado("outro") {
		t.Fatal("chaves diferentes são independentes")
	}
	l.Reiniciar("ip")
	if l.Bloqueado("ip") {
		t.Fatal("reiniciar deveria liberar")
	}
}
