package repositorios

import (
	"orby/api/src/modelos"
	"regexp"
	"strings"
)

// condicoesDeSugestao exclui contas seguidas, silenciadas, bloqueadas, suspensas ou pendentes.
const condicoesDeSugestao = `
	u.id <> ?
	and u.email_verificado_em is not null
	and u.suspenso_em is null
	and not exists (select 1 from seguidores x where x.usuario_id = u.id and x.seguidor_id = ?)
	and not exists (select 1 from pedidos_seguir x where x.alvo_id = u.id and x.solicitante_id = ?)
	and not exists (select 1 from silenciados x where x.silenciador_id = ? and x.silenciado_id = u.id)
	and not exists (
		select 1 from bloqueios b
		where (b.bloqueador_id = ? and b.bloqueado_id = u.id) or (b.bloqueador_id = u.id and b.bloqueado_id = ?)
	)`

// SugerirSeguir prioriza conexões em comum e completa com interações públicas e popularidade.
func (repositorio usuarios) SugerirSeguir(usuarioID uint64, limite int) ([]modelos.SugestaoDeSeguir, error) {
	sugestoes := []modelos.SugestaoDeSeguir{}

	linhas, erro := repositorio.db.Query(`
		select u.id, u.nome, u.nick, u.privado, `+colunaFotoVersao+`, d.comum, d.via
		from usuarios u
		inner join (
			select s2.usuario_id id, count(*) comum, min(m.nick) via
			from seguidores s1
			inner join seguidores s2 on s2.seguidor_id = s1.usuario_id
			inner join usuarios m on m.id = s1.usuario_id
			where s1.seguidor_id = ?
			group by s2.usuario_id
		) d on d.id = u.id
		left join fotos_usuarios f on f.usuario_id = u.id
		where `+condicoesDeSugestao+`
		order by d.comum desc, u.id desc
		limit ?`, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, limite,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	for linhas.Next() {
		var s modelos.SugestaoDeSeguir
		if erro = linhas.Scan(&s.ID, &s.Nome, &s.Nick, &s.Privado, &s.FotoVersao, &s.EmComum, &s.Via); erro != nil {
			return nil, erro
		}
		sugestoes = append(sugestoes, s)
	}
	if erro = linhas.Err(); erro != nil {
		return nil, erro
	}
	linhas.Close()

	if len(sugestoes) >= limite {
		return sugestoes, nil
	}

	tags, erro := repositorio.hashtagsDeInteresse(usuarioID)
	if erro != nil {
		return nil, erro
	}
	pontuacaoTags := "0"
	args := []interface{}{}
	if len(tags) > 0 {
		condicoes := make([]string, 0, len(tags))
		for _, tag := range tags {
			condicoes = append(condicoes, "(pt.titulo regexp ? or pt.conteudo regexp ?)")
			padrao := `(^|[^[:alnum:]_])#` + regexp.QuoteMeta(tag) + `($|[^[:alnum:]_])`
			args = append(args, padrao, padrao)
		}
		pontuacaoTags = `case when u.privado = 0 then (
			select count(*) from publicacoes pt
			where pt.autor_id = u.id and pt.criadoEm >= now() - interval 90 day
			and not exists (select 1 from publicacoes_felas pf where pf.publicacao_id = pt.id)
			and (` + strings.Join(condicoes, " or ") + `)
		) else 0 end`
	}
	args = append(args, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, limite-len(sugestoes))
	// Interesses e interações só contam em publicações de perfis públicos e sem audiência restrita.
	linhas, erro = repositorio.db.Query(`
		select u.id, u.nome, u.nick, u.privado, `+colunaFotoVersao+`,
			`+pontuacaoTags+` as interesses,
			case when u.privado = 0 then (
				(select count(*) from curtidas_publicacoes c inner join publicacoes p on p.id = c.publicacao_id where p.autor_id = u.id and c.usuario_id = ?
					and not exists (select 1 from publicacoes_felas pf where pf.publicacao_id = p.id))
				+ (select count(*) from comentarios c inner join publicacoes p on p.id = c.publicacao_id where p.autor_id = u.id and c.autor_id = ?
					and not exists (select 1 from publicacoes_felas pf where pf.publicacao_id = p.id))
			) else 0 end as interacoes
		from usuarios u
		left join fotos_usuarios f on f.usuario_id = u.id
		where `+condicoesDeSugestao+`
		and not exists (
			select 1 from seguidores s1
			inner join seguidores s2 on s2.seguidor_id = s1.usuario_id
			where s1.seguidor_id = ? and s2.usuario_id = u.id
		)
		order by interesses desc, interacoes desc,
		(select count(*) from seguidores x where x.usuario_id = u.id) desc, u.id desc
		limit ?`, args...,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	for linhas.Next() && len(sugestoes) < limite {
		var s modelos.SugestaoDeSeguir
		var interesses, interacoes int
		if erro = linhas.Scan(&s.ID, &s.Nome, &s.Nick, &s.Privado, &s.FotoVersao, &interesses, &interacoes); erro != nil {
			return nil, erro
		}
		if interesses > 0 {
			s.Motivo = "interesses"
		} else if interacoes > 0 {
			s.Motivo = "interacoes"
		}
		sugestoes = append(sugestoes, s)
	}

	return sugestoes, linhas.Err()
}
