package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/eventos"
	"orby/api/src/modelos"
	"orby/api/src/respostas"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func Acessibilidade(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	db, err := banco.Conectar()
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Close()
	if r.Method == http.MethodGet {
		var tamanho string
		var reduzir bool
		err = db.QueryRow(`select coalesce((select tamanho_texto from preferencias_configuracoes where usuario_id = ?), 'normal'),
			coalesce((select reduzir_animacoes from preferencias_configuracoes where usuario_id = ?), false)`, eu, eu).Scan(&tamanho, &reduzir)
		if err == nil {
			w.Header().Set("Cache-Control", "no-store")
			respostas.JSON(w, http.StatusOK, map[string]interface{}{"tamanho": tamanho, "reduzirAnimacoes": reduzir})
			return
		}
	} else {
		var dados struct {
			Tamanho string `json:"tamanho"`
			Reduzir *bool  `json:"reduzirAnimacoes"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&dados) != nil || dados.Reduzir == nil ||
			(dados.Tamanho != "normal" && dados.Tamanho != "grande" && dados.Tamanho != "muito-grande") {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Preferência inválida"))
			return
		}
		_, err = db.Exec(`insert into preferencias_configuracoes (usuario_id, tamanho_texto, reduzir_animacoes) values (?, ?, ?)
			on duplicate key update tamanho_texto = values(tamanho_texto), reduzir_animacoes = values(reduzir_animacoes)`, eu, dados.Tamanho, *dados.Reduzir)
		if err == nil {
			respostas.JSON(w, http.StatusNoContent, nil)
			return
		}
	}
	respostas.Erro(w, http.StatusInternalServerError, err)
}

func PausaDeNotificacoes(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	db, err := banco.Conectar()
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Close()
	if r.Method == http.MethodGet {
		var permanente bool
		var segundos int64
		err = db.QueryRow(`select coalesce((select pausa_permanente from preferencias_configuracoes where usuario_id = ?), false),
			coalesce((select greatest(0, timestampdiff(second, utc_timestamp(), pausa_ate)) from preferencias_configuracoes where usuario_id = ?), 0)`, eu, eu).Scan(&permanente, &segundos)
		if err == nil {
			w.Header().Set("Cache-Control", "no-store")
			respostas.JSON(w, http.StatusOK, map[string]interface{}{"permanente": permanente, "segundosRestantes": segundos})
			return
		}
	} else {
		var dados struct {
			Duracao string `json:"duracao"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&dados) != nil {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Pausa inválida"))
			return
		}
		horas, valido := map[string]int{"retomar": 0, "1h": 1, "8h": 8, "24h": 24, "permanente": 0}[dados.Duracao]
		if !valido {
			respostas.Erro(w, http.StatusBadRequest, errors.New("Pausa inválida"))
			return
		}
		_, err = db.Exec(`insert into preferencias_configuracoes (usuario_id, pausa_permanente, pausa_ate)
			values (?, ?, if(? = 0, null, date_add(utc_timestamp(), interval ? hour)))
			on duplicate key update pausa_permanente = values(pausa_permanente), pausa_ate = values(pausa_ate)`,
			eu, dados.Duracao == "permanente", horas, horas)
		if err == nil {
			eventos.Publicar(eu, eventos.Evento{Tipo: "notificacao"})
			respostas.JSON(w, http.StatusNoContent, nil)
			return
		}
	}
	respostas.Erro(w, http.StatusInternalServerError, err)
}

func PessoasRestritas(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	db, err := banco.Conectar()
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Close()
	type pessoa struct {
		ID   uint64 `json:"id"`
		Nome string `json:"nome"`
		Nick string `json:"nick"`
	}
	resultado := map[string][]pessoa{"bloqueados": {}, "silenciados": {}}
	for chave, consulta := range map[string]string{
		"bloqueados":  `select u.id, u.nome, u.nick from bloqueios b join usuarios u on u.id = b.bloqueado_id where b.bloqueador_id = ? order by u.nick`,
		"silenciados": `select u.id, u.nome, u.nick from silenciados s join usuarios u on u.id = s.silenciado_id where s.silenciador_id = ? order by u.nick`,
	} {
		linhas, erro := db.Query(consulta, eu)
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
		for linhas.Next() {
			var p pessoa
			if erro = linhas.Scan(&p.ID, &p.Nome, &p.Nick); erro != nil {
				break
			}
			resultado[chave] = append(resultado[chave], p)
		}
		if erro == nil {
			erro = linhas.Err()
		}
		linhas.Close()
		if erro != nil {
			respostas.Erro(w, http.StatusInternalServerError, erro)
			return
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	respostas.JSON(w, http.StatusOK, resultado)
}

func DadosDaConta(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	var dados struct {
		Email string `json:"email"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&dados) != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("E-mail inválido"))
		return
	}
	dados.Email = strings.TrimSpace(dados.Email)
	validar := modelos.Usuario{Nome: "x", Nick: "x", Email: dados.Email}
	if err := validar.Preparar("edicao"); err != nil {
		respostas.Erro(w, http.StatusBadRequest, err)
		return
	}
	db, err := banco.Conectar()
	if err == nil {
		defer db.Close()
		_, err = db.Exec("update usuarios set email = ? where id = ?", validar.Email, eu)
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		respostas.Erro(w, http.StatusConflict, errors.New("E-mail já cadastrado"))
		return
	}
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}

func DadosDoPerfil(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	var usuario modelos.Usuario
	if json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&usuario) != nil {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Perfil inválido"))
		return
	}
	usuario.Email = "validacao@orby.test"
	if err := usuario.Preparar("edicao"); err != nil {
		respostas.Erro(w, http.StatusBadRequest, err)
		return
	}
	db, err := banco.Conectar()
	if err == nil {
		defer db.Close()
		_, err = db.Exec("update usuarios set nome = ?, nick = ?, bio = ? where id = ?", usuario.Nome, usuario.Nick, usuario.Bio, eu)
	}
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		respostas.Erro(w, http.StatusConflict, errors.New("Nick já cadastrado"))
		return
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	respostas.JSON(w, http.StatusNoContent, nil)
}
