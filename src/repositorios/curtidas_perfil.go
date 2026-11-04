package repositorios

import (
	"database/sql"
	"orby/api/src/modelos"
)

type CurtidasPerfil struct {
	Total        uint64               `json:"total"`
	TotalVisivel bool                 `json:"totalVisivel"`
	ListaPublica bool                 `json:"listaPublica"`
	PodeVer      bool                 `json:"podeVer"`
	Publicacoes  []modelos.Publicacao `json:"publicacoes"`
}

func PrivacidadeCurtidas(db *sql.DB, usuarioID uint64) (bool, error) {
	var publica bool
	erro := db.QueryRow("select curtidas_publicas from usuarios where id = ?", usuarioID).Scan(&publica)
	return publica, erro
}

func SalvarPrivacidadeCurtidas(db *sql.DB, usuarioID uint64, publica bool) error {
	_, erro := db.Exec("update usuarios set curtidas_publicas = ? where id = ?", publica, usuarioID)
	return erro
}

func (repositorio Publicacoes) CurtidasDoPerfil(alvo, visitante uint64, listar bool) (CurtidasPerfil, error) {
	resultado := CurtidasPerfil{Publicacoes: []modelos.Publicacao{}}
	var publica, acessoPerfil bool
	erro := repositorio.db.QueryRow(`
		select curtidas_publicas, (u.id = ? or (
			(u.privado = 0 or exists(select 1 from seguidores s where s.usuario_id = u.id and s.seguidor_id = ?))
			and not exists(select 1 from bloqueios b where
				(b.bloqueador_id = u.id and b.bloqueado_id = ?) or (b.bloqueador_id = ? and b.bloqueado_id = u.id))))
		from usuarios u where u.id = ?`, visitante, visitante, visitante, visitante, alvo).Scan(&publica, &acessoPerfil)
	if erro != nil {
		return resultado, erro
	}
	resultado.TotalVisivel = acessoPerfil
	resultado.ListaPublica = acessoPerfil && publica
	resultado.PodeVer = acessoPerfil && (alvo == visitante || publica)
	if !acessoPerfil {
		return resultado, nil
	}
	// A contagem só inclui publicações visíveis ao visitante, inclusive para Felas.
	erro = repositorio.db.QueryRow(`
		select count(*) from curtidas_publicacoes cp
		join publicacoes p on p.id = cp.publicacao_id
		join usuarios u on u.id = p.autor_id
		where p.autor_id = ? and `+postVisivel,
		alvo, visitante, visitante, visitante, visitante).Scan(&resultado.Total)
	if erro != nil || !listar || !resultado.PodeVer {
		return resultado, erro
	}
	linhas, erro := repositorio.db.Query(`
		select p.*, u.nick, u.suspenso_em is not null, `+colunaFotoVersao+`,
			(select count(*) from comentarios c where c.publicacao_id = p.id),
			`+colunaTemImagem+`,
			exists(select 1 from publicacoes_felas pf where pf.publicacao_id = p.id),
			exists(select 1 from curtidas_publicacoes c where c.publicacao_id = p.id and c.usuario_id = ?),
			exists(select 1 from salvos sv where sv.publicacao_id = p.id and sv.usuario_id = ?)
		from curtidas_publicacoes cp
		join publicacoes p on p.id = cp.publicacao_id
		join usuarios u on u.id = p.autor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where cp.usuario_id = ? and `+postVisivel+`
			and not exists(select 1 from bloqueios b where
				(b.bloqueador_id = u.id and b.bloqueado_id = ?) or (b.bloqueador_id = ? and b.bloqueado_id = u.id))
		order by p.id desc`, visitante, visitante, alvo, visitante, visitante, visitante, visitante, visitante, visitante)
	if erro != nil {
		return resultado, erro
	}
	defer linhas.Close()
	for linhas.Next() {
		var p modelos.Publicacao
		var editada sql.NullTime
		if erro = linhas.Scan(&p.ID, &p.Titulo, &p.Conteudo, &p.AutorID, &p.Curtidas, &p.CriadaEm, &editada,
			&p.AutorNick, &p.AutorSuspenso, &p.AutorFotoVersao, &p.Comentarios, &p.TemImagem, &p.Felas,
			&p.CurtidoPorUsuario, &p.SalvoPorUsuario); erro != nil {
			return resultado, erro
		}
		p.Editado = editada.Valid
		resultado.Publicacoes = append(resultado.Publicacoes, p)
	}
	if erro = linhas.Err(); erro != nil {
		return resultado, erro
	}
	if erro = linhas.Close(); erro != nil {
		return resultado, erro
	}
	if erro = repositorio.AnexarRepostagens(resultado.Publicacoes, visitante); erro != nil {
		return resultado, erro
	}
	return resultado, nil
}
