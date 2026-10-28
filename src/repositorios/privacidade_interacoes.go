package repositorios

import (
	"database/sql"
	"errors"
)

var ErrPrivacidadeInteracao = errors.New("Esta pessoa restringiu esta interação nas configurações de privacidade")
var ErrPreferenciaPrivacidadeInvalida = errors.New("Preferência de privacidade inválida")

type PrivacidadeInteracoes struct {
	Mensagens   string `json:"mensagens"`
	Mencoes     string `json:"mencoes"`
	Comentarios string `json:"comentarios"`
}

func PreferenciasDePrivacidade(db *sql.DB, usuarioID uint64) (PrivacidadeInteracoes, error) {
	p := PrivacidadeInteracoes{Mensagens: "todos", Mencoes: "todos", Comentarios: "todos"}
	erro := db.QueryRow("select mensagens, mencoes, comentarios from preferencias_privacidade where usuario_id = ?", usuarioID).Scan(&p.Mensagens, &p.Mencoes, &p.Comentarios)
	if errors.Is(erro, sql.ErrNoRows) {
		return p, nil
	}
	return p, erro
}

func SalvarPrivacidade(db *sql.DB, usuarioID uint64, p PrivacidadeInteracoes) error {
	valida := func(valor string) bool { return valor == "todos" || valor == "seguindo" || valor == "ninguem" }
	if !valida(p.Mensagens) || !valida(p.Mencoes) || !valida(p.Comentarios) {
		return ErrPreferenciaPrivacidadeInvalida
	}
	tx, erro := db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	if erro = travarUsuarios(tx, usuarioID, usuarioID); erro != nil {
		return erro
	}
	_, erro = tx.Exec(`insert into preferencias_privacidade (usuario_id, mensagens, mencoes, comentarios) values (?, ?, ?, ?)
		on duplicate key update mensagens = values(mensagens), mencoes = values(mencoes), comentarios = values(comentarios)`,
		usuarioID, p.Mensagens, p.Mencoes, p.Comentarios)
	if erro != nil {
		return erro
	}
	return tx.Commit()
}

func PermiteInteracao(consulta consultador, alvo, ator uint64, tipo string) (bool, error) {
	if alvo == ator {
		return true, nil
	}
	var coluna string
	switch tipo {
	case "mensagens", "mencoes", "comentarios":
		coluna = tipo
	default:
		return false, ErrPreferenciaPrivacidadeInvalida
	}
	var permite bool
	erro := consulta.QueryRow(`
		select coalesce(pr.`+coluna+`, 'todos') = 'todos'
		or (pr.`+coluna+` = 'seguindo'
			and exists(select 1 from seguidores where usuario_id = ? and seguidor_id = u.id))
		from usuarios u left join preferencias_privacidade pr on pr.usuario_id = u.id
		where u.id = ? for update`, ator, alvo).Scan(&permite)
	return permite, erro
}
