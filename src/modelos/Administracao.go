package modelos

import "time"

// ItemAdministrativo é uma linha das listagens da administração.
type ItemAdministrativo struct {
	ID            uint64    `json:"id"`
	UsuarioID     uint64    `json:"usuarioId"`
	Nome          string    `json:"nome"`
	Nick          string    `json:"nick"`
	Titulo        string    `json:"titulo"`
	Conteudo      string    `json:"conteudo"`
	Data          time.Time `json:"data"`
	Total         uint64    `json:"total"`
	Suspenso      bool      `json:"suspenso"`
	AlvoPrivadoID uint64    `json:"-"`
}
