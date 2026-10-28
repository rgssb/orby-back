package repositorios

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var hashtagDeInteresse = regexp.MustCompile(`(?:^|[^\p{L}\p{N}_@#])#([\p{L}\p{N}_]+)`)

func (repositorio usuarios) hashtagsDeInteresse(usuarioID uint64) ([]string, error) {
	linhas, erro := repositorio.db.Query(`
		select p.titulo, p.conteudo from publicacoes p
		join usuarios u on u.id = p.autor_id
		where u.privado = 0 and u.suspenso_em is null
			and p.criadoEm >= now() - interval 90 day
			and not exists (select 1 from publicacoes_felas pf where pf.publicacao_id = p.id)
			and not exists (select 1 from silenciados sl where sl.silenciador_id = ? and sl.silenciado_id = u.id)
			and not exists (select 1 from bloqueios b where
				(b.bloqueador_id = ? and b.bloqueado_id = u.id) or
				(b.bloqueador_id = u.id and b.bloqueado_id = ?))
			and (p.autor_id = ? or exists (select 1 from seguidores s where s.usuario_id = p.autor_id and s.seguidor_id = ?)
				or exists (select 1 from curtidas_publicacoes cp where cp.publicacao_id = p.id and cp.usuario_id = ?)
				or exists (select 1 from comentarios c where c.publicacao_id = p.id and c.autor_id = ?))
		order by p.id desc limit 200`, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID, usuarioID)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	frequencias := map[string]int{}
	for linhas.Next() {
		var titulo, conteudo string
		if erro = linhas.Scan(&titulo, &conteudo); erro != nil {
			return nil, erro
		}
		vistas := map[string]bool{}
		for _, achado := range hashtagDeInteresse.FindAllStringSubmatch(titulo+"\n"+conteudo, -1) {
			tag := strings.ToLower(achado[1])
			if utf8.RuneCountInString(tag) <= 50 && !vistas[tag] {
				frequencias[tag]++
				vistas[tag] = true
			}
		}
	}
	if erro = linhas.Err(); erro != nil {
		return nil, erro
	}
	tags := make([]string, 0, len(frequencias))
	for tag := range frequencias {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool {
		if frequencias[tags[i]] == frequencias[tags[j]] {
			return tags[i] < tags[j]
		}
		return frequencias[tags[i]] > frequencias[tags[j]]
	})
	if len(tags) > 5 {
		tags = tags[:5]
	}
	return tags, nil
}
