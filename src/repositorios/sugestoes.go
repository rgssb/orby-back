package repositorios

import (
	"orby/api/src/modelos"
)

// condicoesDeSugestao exclui quem o usuário já segue, já pediu para seguir, tem bloqueio com ele ou não confirmou o e-mail.
// O usuário aparece quatro vezes, na ordem: ele mesmo, seguindo, pedido, bloqueios (2x).
const condicoesDeSugestao = `
	u.id <> ?
	and u.email_verificado_em is not null
	and not exists (select 1 from seguidores x where x.usuario_id = u.id and x.seguidor_id = ?)
	and not exists (select 1 from pedidos_seguir x where x.alvo_id = u.id and x.solicitante_id = ?)
	and not exists (
		select 1 from bloqueios b
		where (b.bloqueador_id = ? and b.bloqueado_id = u.id) or (b.bloqueador_id = u.id and b.bloqueado_id = ?)
	)`

// SugerirSeguir traz pessoas que quem o usuário segue também segue, das mais às menos em comum.
// Se sobrar espaço, completa com as pessoas mais seguidas da rede (EmComum = 0).
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
		limit ?`, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, limite,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	jaSugeridos := map[uint64]bool{}
	for linhas.Next() {
		var s modelos.SugestaoDeSeguir
		if erro = linhas.Scan(&s.ID, &s.Nome, &s.Nick, &s.Privado, &s.FotoVersao, &s.EmComum, &s.Via); erro != nil {
			return nil, erro
		}
		jaSugeridos[s.ID] = true
		sugestoes = append(sugestoes, s)
	}
	if erro = linhas.Err(); erro != nil {
		return nil, erro
	}
	linhas.Close()

	if len(sugestoes) >= limite {
		return sugestoes, nil
	}

	// Preenche com as pessoas mais seguidas; as já sugeridas são puladas aqui no Go
	linhas, erro = repositorio.db.Query(`
		select u.id, u.nome, u.nick, u.privado, `+colunaFotoVersao+`
		from usuarios u
		left join fotos_usuarios f on f.usuario_id = u.id
		where `+condicoesDeSugestao+`
		order by (select count(*) from seguidores x where x.usuario_id = u.id) desc, u.id desc
		limit ?`, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, limite,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	for linhas.Next() && len(sugestoes) < limite {
		var s modelos.SugestaoDeSeguir
		if erro = linhas.Scan(&s.ID, &s.Nome, &s.Nick, &s.Privado, &s.FotoVersao); erro != nil {
			return nil, erro
		}
		if !jaSugeridos[s.ID] {
			sugestoes = append(sugestoes, s)
		}
	}

	return sugestoes, linhas.Err()
}
