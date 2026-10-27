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
