package demonstracao

import (
	"database/sql"
	"errors"
	"fmt"
)

// AtualizarTextosAntigos substitui apenas textos originais conhecidos da versão anterior.
func AtualizarTextosAntigos(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, assunto := range assuntos {
		nick := fmt.Sprintf("usuario_%d", i+1)
		email := fmt.Sprintf("usuario%d@gmail.com", i+1)
		var post uint64
		err = tx.QueryRow(`select p.id from publicacoes p join usuarios u on u.id = p.autor_id
			where u.nick = ? and u.email = ? and binary p.titulo = ? and binary p.conteudo = ?`,
			nick, email, fmt.Sprintf("Publicação de exemplo %d", i+1),
			fmt.Sprintf("Olá! Sou Usuario%d. Esta publicação é um exemplo para conhecer o Orby. #exemplo", i+1)).Scan(&post)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if i < 8 {
			ator := fmt.Sprintf("usuario_%d", (i^1)+1)
			if _, err = tx.Exec(`update comentarios resposta
				join comentarios comentario on comentario.id = resposta.parent_id
				join usuarios autor on autor.id = resposta.autor_id
				join usuarios outro on outro.id = comentario.autor_id
				set resposta.conteudo = ?
				where resposta.publicacao_id = ? and autor.nick = ? and outro.nick = ?
				and binary resposta.conteudo = ? and binary comentario.conteudo = ?`,
				assunto.resposta, post, nick, ator, "Obrigado! Esta é uma resposta de exemplo.", "Comentário de exemplo: gostei da publicação!"); err != nil {
				return err
			}
			if _, err = tx.Exec(`update comentarios c join usuarios u on u.id = c.autor_id
				set c.conteudo = ? where c.publicacao_id = ? and u.nick = ? and c.parent_id is null and binary c.conteudo = ?`,
				assunto.comentario, post, ator, "Comentário de exemplo: gostei da publicação!"); err != nil {
				return err
			}
		}
		if _, err = tx.Exec("update publicacoes set titulo = ?, conteudo = ? where id = ?", assunto.titulo, assunto.conteudo, post); err != nil {
			return err
		}
	}
	for _, troca := range []struct {
		consulta string
		args     []any
	}{
		{`insert into mensagens (conversa_id, remetente_id, texto)
			select c.id, destinatario.id, 'Oi! Tudo bem sim. Vamos marcar aquele café esta semana?'
			from conversas c join usuarios remetente on remetente.id = c.iniciada_por
			join usuarios destinatario on destinatario.nick = 'usuario_2' and destinatario.email = 'usuario2@gmail.com'
			where remetente.nick = 'usuario_1' and remetente.email = 'usuario1@gmail.com'
			and c.usuario_menor = least(remetente.id, destinatario.id) and c.usuario_maior = greatest(remetente.id, destinatario.id)
			and (select count(*) from mensagens m where m.conversa_id = c.id) = 1
			and exists(select 1 from mensagens m where m.conversa_id = c.id and m.remetente_id = remetente.id
				and (binary m.texto = 'Olá! Esta é uma mensagem de exemplo.'
					or binary m.texto = 'Oi! Como você está? Faz tempo que não conversamos.'))`, nil},
		{`insert into grupo_mensagens (grupo_id, remetente_id, texto)
			select g.id, membro.id, 'Topo! Sábado à tarde funciona para vocês?'
			from grupos g join usuarios criador on criador.id = g.criador_id
			join usuarios membro on membro.nick = 'usuario_1' and membro.email = 'usuario1@gmail.com'
			join grupo_membros gm on gm.grupo_id = g.id and gm.usuario_id = membro.id
			where criador.nick = 'usuario_2' and criador.email = 'usuario2@gmail.com'
			and (binary g.nome = 'Grupo de exemplo' or binary g.nome = 'Papo de fim de tarde')
			and (select count(*) from grupo_mensagens m where m.grupo_id = g.id) = 1
			and exists(select 1 from grupo_mensagens m where m.grupo_id = g.id and m.remetente_id = criador.id
				and (binary m.texto = 'Olá! Este grupo é um exemplo para testar conversas.'
					or binary m.texto = 'Oi, pessoal! O que vocês acham de marcar um café no fim de semana?'))`, nil},
		{`update publicacoes p join usuarios u on u.id = p.autor_id join publicacoes_felas f on f.publicacao_id = p.id
			set p.conteudo = ? where u.nick = 'usuario_1' and u.email = 'usuario1@gmail.com' and binary p.titulo = ? and binary p.conteudo = ?`,
			[]any{"Passando para agradecer por estarem por perto. Vocês deixam meus dias mais leves!", "Para meus Felas", "Esta é uma publicação de exemplo visível para meus Felas."}},
		{`update opcoes_enquetes o join publicacoes p on p.id = o.publicacao_id join usuarios u on u.id = p.autor_id
			set o.texto = case when binary o.texto = 'Publicações' then 'Passeio ao ar livre' else 'Filme em casa' end
			where u.nick = 'usuario_2' and u.email = 'usuario2@gmail.com' and binary p.titulo = 'Enquete de exemplo'
			and binary p.conteudo = 'Qual recurso do Orby você mais gosta?' and (binary o.texto = 'Publicações' or binary o.texto = 'Mensagens')`, nil},
		{`update enquetes e join publicacoes p on p.id = e.publicacao_id join usuarios u on u.id = p.autor_id
			set e.titulo = 'Escolha seu favorito' where u.nick = 'usuario_2' and u.email = 'usuario2@gmail.com'
			and binary p.titulo = 'Enquete de exemplo' and binary p.conteudo = 'Qual recurso do Orby você mais gosta?' and binary e.titulo = 'Escolha uma opção'`, nil},
		{`update publicacoes p join usuarios u on u.id = p.autor_id join enquetes e on e.publicacao_id = p.id
			set p.titulo = 'Qual é o plano?', p.conteudo = 'Como vocês preferem passar uma tarde livre?'
			where u.nick = 'usuario_2' and u.email = 'usuario2@gmail.com' and binary p.titulo = 'Enquete de exemplo'
			and binary p.conteudo = 'Qual recurso do Orby você mais gosta?'`, nil},
		{`update publicacoes p join usuarios u on u.id = p.autor_id join reposts r on r.publicacao_id = p.id
			set p.conteudo = 'Café e boas ideias combinam demais!' where u.nick = 'usuario_2' and u.email = 'usuario2@gmail.com'
			and binary p.conteudo = 'Repost de exemplo'`, nil},
		{`update pastas_salvos p join usuarios u on u.id = p.usuario_id set p.nome = 'Para guardar'
			where u.nick = 'usuario_2' and u.email = 'usuario2@gmail.com' and binary p.nome = 'Exemplos'`, nil},
		{`update mensagens m join usuarios u on u.id = m.remetente_id
			set m.texto = 'Oi! Como você está? Faz tempo que não conversamos.'
			where u.nick = 'usuario_1' and u.email = 'usuario1@gmail.com' and binary m.texto = 'Olá! Esta é uma mensagem de exemplo.'`, nil},
		{`update grupo_mensagens m join grupos g on g.id = m.grupo_id join usuarios u on u.id = g.criador_id
			set m.texto = 'Oi, pessoal! O que vocês acham de marcar um café no fim de semana?'
			where m.remetente_id = g.criador_id and u.nick = 'usuario_2' and u.email = 'usuario2@gmail.com'
			and binary g.nome = 'Grupo de exemplo' and binary m.texto = 'Olá! Este grupo é um exemplo para testar conversas.'`, nil},
		{`update grupos g join usuarios u on u.id = g.criador_id set g.nome = 'Papo de fim de tarde'
			where u.nick = 'usuario_2' and u.email = 'usuario2@gmail.com' and binary g.nome = 'Grupo de exemplo'`, nil},
	} {
		if _, err = tx.Exec(troca.consulta, troca.args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}
