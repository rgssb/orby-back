package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"orby/api/src/autenticacao"
	"orby/api/src/banco"
	"orby/api/src/respostas"
	"regexp"
)

var nickDeMencao = regexp.MustCompile(`^[A-Za-z0-9_]{1,50}$`)

func VisibilidadeDasMencoes(w http.ResponseWriter, r *http.Request) {
	eu, erro := autenticacao.ExtrairUsuarioID(r)
	if erro != nil {
		respostas.Erro(w, http.StatusUnauthorized, erro)
		return
	}
	var corpo struct {
		Nicks []string `json:"nicks"`
	}
	if erro = json.NewDecoder(r.Body).Decode(&corpo); erro != nil || len(corpo.Nicks) > 100 {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Lista de menções inválida"))
		return
	}
	for _, nick := range corpo.Nicks {
		if !nickDeMencao.MatchString(nick) {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Menção inválida"))
			return
		}
	}
	db, erro := banco.Conectar()
	if erro != nil {
		respostas.Erro(w, http.StatusInternalServerError, erro)
		return
	}
	defer db.Close()
	resultado := make(map[string]bool, len(corpo.Nicks))
	for _, nick := range corpo.Nicks {
		var visivel bool
		erro = db.QueryRow(`
			select u.id = ? or coalesce(pr.mencoes, 'todos') = 'todos'
				or (pr.mencoes = 'seguindo' and (
					exists(select 1 from seguidores where usuario_id = u.id and seguidor_id = ?)
					or exists(select 1 from seguidores where usuario_id = ? and seguidor_id = u.id)))
			from usuarios u left join preferencias_privacidade pr on pr.usuario_id = u.id
			where replace(u.nick, ' ', '') = ? order by u.nick = ? desc limit 1`,
			eu, eu, eu, nick, nick).Scan(&visivel)
		if erro != nil && !errors.Is(erro, sql.ErrNoRows) {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		resultado[nick] = visivel
	}
	respostas.JSON(w, http.StatusOK, resultado)
}
