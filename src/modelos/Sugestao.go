package modelos

// Sugestao é um usuário sugerido no autocomplete de @menção.
type Sugestao struct {
	ID         uint64 `json:"id"`
	Nick       string `json:"nick"`
	FotoVersao uint64 `json:"fotoVersao"`
}
