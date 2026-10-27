package modelos

import "time"

// Tipos de conteúdo que podem ser denunciados
const (
	DenunciaPerfil     = "perfil"
	DenunciaPublicacao = "publicacao"
	DenunciaComentario = "comentario"
)

// PerfilDenunciado resume as denúncias abertas contra uma pessoa, para a revisão da administração
type PerfilDenunciado struct {
	UsuarioID   uint64         `json:"usuarioId"`
	Nick        string         `json:"nick"`
	Nome        string         `json:"nome"`
	Total       uint64         `json:"total"`
	Nivel       string         `json:"nivel"`
	Perfis      uint64         `json:"perfis"`
	Publicacoes uint64         `json:"publicacoes"`
	Comentarios uint64         `json:"comentarios"`
	Motivos     map[string]int `json:"motivos"`
	UltimaEm    time.Time      `json:"ultimaEm"`
}

// ContaSuspensa é uma conta barrada pela administração
type ContaSuspensa struct {
	UsuarioID  uint64    `json:"usuarioId"`
	Nome       string    `json:"nome"`
	Nick       string    `json:"nick"`
	SuspensaEm time.Time `json:"suspensaEm"`
}

// EstatisticasDoOrby são os números do painel da administração
type EstatisticasDoOrby struct {
	Usuarios          uint64 `json:"usuarios"`
	Publicacoes       uint64 `json:"publicacoes"`
	Comentarios       uint64 `json:"comentarios"`
	Suspensos         uint64 `json:"suspensos"`
	PerfisDenunciados uint64 `json:"perfisDenunciados"`
	NovosNaSemana     uint64 `json:"novosNaSemana"`
}
