package modelos

import "time"

// Tipos de notificação
const (
	NotificacaoCurtidaPublicacao = "curtida_publicacao"
	NotificacaoComentario        = "comentario"
	NotificacaoCurtidaComentario = "curtida_comentario"
	NotificacaoResposta          = "resposta"
	NotificacaoRepost            = "repost"
	NotificacaoMencao            = "mencao"
	NotificacaoSeguidor          = "seguidor"
	NotificacaoPedidoSeguir      = "pedido_seguir"
)

// Notificacao avisa o dono de uma publicação ou comentário sobre uma interação de outro usuário.
type Notificacao struct {
	ID               uint64    `json:"id"`
	Tipo             string    `json:"tipo"`
	AtorID           uint64    `json:"atorId"`
	AtorNick         string    `json:"atorNick"`
	AtorFotoVersao   uint64    `json:"atorFotoVersao"`
	PublicacaoID     uint64    `json:"publicacaoId"`
	PublicacaoTitulo string    `json:"publicacaoTitulo"`
	ComentarioID     uint64    `json:"comentarioId,omitempty"`
	Lida             bool      `json:"lida"`
	CriadaEm         time.Time `json:"criadaEm"`
}
