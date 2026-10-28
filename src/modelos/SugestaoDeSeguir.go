package modelos

// SugestaoDeSeguir é uma pessoa sugerida para o usuário seguir.
// EmComum é quantas pessoas que o usuário segue também seguem essa pessoa e Via é uma delas.
type SugestaoDeSeguir struct {
	ID         uint64 `json:"id"`
	Nome       string `json:"nome"`
	Nick       string `json:"nick"`
	FotoVersao uint64 `json:"fotoVersao"`
	Privado    bool   `json:"privado"`
	EmComum    int    `json:"emComum"`
	Via        string `json:"via"`
	Motivo     string `json:"motivo,omitempty"`
}
