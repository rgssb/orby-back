// Package demonstracao cria dados fictícios apenas quando solicitado.
package demonstracao

import (
	"database/sql"
	"errors"
	"fmt"
	"orby/api/src/modelos"
	"orby/api/src/seguranca"
)

type conversaPost struct{ titulo, conteudo, comentario, resposta string }

var assuntos = []conversaPost{
	{"Café e ideias", "Comecei o dia com café e uma lista enorme de ideias. Qual é o ritual de vocês para entrar no ritmo?", "Café e uma playlist calma por aqui. Aceito recomendações!", "Boa! Vou montar uma playlist nova para amanhã."},
	{"Fim de tarde", "O céu estava tão bonito hoje que precisei parar um pouco para olhar. Esses momentos simples fazem diferença.", "Também vi! A luz estava linda depois da chuva.", "Sim! Queria que o pôr do sol durasse mais."},
	{"Livro da semana", "Terminei um livro que estava na minha lista há meses. Agora estou procurando minha próxima leitura, alguma indicação?", "Tenho uma indicação! Você prefere ficção ou mistério?", "Mistério, sem dúvida. Pode mandar!"},
	{"Música para trabalhar", "Preciso renovar minha playlist de trabalho. O que vocês ouvem quando precisam se concentrar?", "Música instrumental me ajuda muito. Posso compartilhar algumas faixas.", "Quero sim! Instrumental funciona bem para mim também."},
	{"Pequenas conquistas", "Hoje consegui terminar uma tarefa que estava adiando havia semanas. Parece pouco, mas foi uma vitória!", "Nada pouco! Essas tarefas pendentes cansam bastante.", "Verdade. A sensação de tirar isso da lista é ótima."},
	{"Receita improvisada", "Fiz um macarrão com o que tinha na geladeira e ficou melhor do que eu esperava. Adoro quando a improvisação dá certo.", "Agora fiquei com fome. O que você colocou no molho?", "Tomate, alho e um pouco de queijo. Simples e gostoso!"},
	{"Passeio de fim de semana", "Quero sair para caminhar no fim de semana e conhecer um lugar novo. Alguma sugestão de passeio tranquilo?", "Um parque no começo da manhã é sempre uma boa.", "Gostei! Vou separar um tempo para isso."},
	{"Ideias para criar", "Estou tentando desenhar um pouco todos os dias, sem me preocupar tanto com o resultado. Está sendo divertido.", "Essa é a melhor parte! O importante é aproveitar o processo.", "Exatamente. Aos poucos a prática vai ficando mais leve."},
	{"Hora de descansar", "Dia cheio por aqui. Agora só quero uma boa conversa e um tempo longe das notificações.", "Descansar também faz parte. Aproveite a pausa!", "Obrigado! Amanhã continuo com mais energia."},
	{"Planos para amanhã", "Deixei algumas coisas organizadas para amanhã e já estou me sentindo mais tranquilo. Boa noite, pessoal!", "Boa noite! Começar o dia com tudo organizado ajuda muito.", "Ajuda mesmo. Espero que amanhã seja um dia leve!"},
}

// Criar adiciona as contas de exemplo e suas interações em uma transação.
// Perfis existentes só são conectados quando seus nicks são fornecidos.
func Criar(db *sql.DB, senha string, perfis []string) error {
	if err := modelos.ValidarNovaSenha(senha); err != nil {
		return err
	}
	hash, err := seguranca.Hash(senha)
	if err != nil {
		return fmt.Errorf("gerar hash da senha: %w", err)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	existentes := make([]int64, 0, len(perfis))
	vistos := make(map[int64]bool)
	for _, nick := range perfis {
		var id int64
		err = tx.QueryRow("select id from usuarios where nick = ?", nick).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("perfil %q não encontrado", nick)
		}
		if err != nil {
			return err
		}
		if vistos[id] {
			return fmt.Errorf("perfil %q indicado mais de uma vez", nick)
		}
		vistos[id] = true
		existentes = append(existentes, id)
	}

	var usuarios [10]int64
	var posts [10]int64
	for i := range usuarios {
		nome := fmt.Sprintf("Usuario%d", i+1)
		nick := fmt.Sprintf("usuario_%d", i+1)
		email := fmt.Sprintf("usuario%d@gmail.com", i+1)
		var conflitos int
		if err = tx.QueryRow("select count(*) from usuarios where nick = ? or email = ?", nick, email).Scan(&conflitos); err != nil {
			return err
		}
		if conflitos != 0 {
			return fmt.Errorf("nick ou email de %s já está em uso; nenhum dado foi alterado", nick)
		}
		usuarios[i], err = inserir(tx, `insert into usuarios
			(nome, nick, email, senha, bio, privado, termos_aceitos_em, email_verificado_em)
			values (?, ?, ?, ?, ?, ?, now(), now())`,
			nome, nick, email, string(hash), "Conta fictícia para demonstração do Orby.", i == 9)
		if err != nil {
			return fmt.Errorf("criar %s: %w", nick, err)
		}
		posts[i], err = inserir(tx, `insert into publicacoes (titulo, conteudo, autor_id)
			values (?, ?, ?)`, assuntos[i].titulo, assuntos[i].conteudo, usuarios[i])
		if err != nil {
			return fmt.Errorf("criar publicação: %w", err)
		}
	}

	// Pares mútuos, com uma solicitação pendente para a conta privada.
	for i := 0; i < 8; i += 2 {
		if err = seguirMutuamente(tx, usuarios[i], usuarios[i+1]); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("insert into pedidos_seguir (solicitante_id, alvo_id) values (?, ?)", usuarios[8], usuarios[9]); err != nil {
		return err
	}
	for _, perfil := range existentes {
		for i := 0; i < 4; i++ {
			if err = seguirMutuamente(tx, perfil, usuarios[i]); err != nil {
				return err
			}
			if _, err = tx.Exec("insert into felas (dono_id, membro_id) values (?, ?), (?, ?)", perfil, usuarios[i], usuarios[i], perfil); err != nil {
				return err
			}
			if _, err = tx.Exec("insert into notificacoes (usuario_id, ator_id, tipo) values (?, ?, ?)", perfil, usuarios[i], modelos.NotificacaoSeguidor); err != nil {
				return err
			}
		}
		if err = conversa(tx, usuarios[0], perfil); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("insert into felas (dono_id, membro_id) values (?, ?), (?, ?)", usuarios[0], usuarios[1], usuarios[1], usuarios[0]); err != nil {
		return err
	}

	for i := 0; i < 8; i++ {
		ator := usuarios[i^1]
		if _, err = tx.Exec("insert into curtidas_publicacoes (publicacao_id, usuario_id) values (?, ?)", posts[i], ator); err != nil {
			return err
		}
		if _, err = tx.Exec("update publicacoes set curtidas = curtidas + 1 where id = ?", posts[i]); err != nil {
			return err
		}
		comentario, err := inserir(tx, "insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, ?)", posts[i], ator, assuntos[i].comentario)
		if err != nil {
			return err
		}
		if _, err = tx.Exec("insert into comentarios (publicacao_id, autor_id, conteudo, parent_id) values (?, ?, ?, ?)", posts[i], usuarios[i], assuntos[i].resposta, comentario); err != nil {
			return err
		}
		if _, err = tx.Exec("insert into curtidas_comentarios (comentario_id, usuario_id) values (?, ?)", comentario, usuarios[i]); err != nil {
			return err
		}
		if _, err = tx.Exec("insert into reacoes_publicacoes (publicacao_id, usuario_id, emoji) values (?, ?, ?)", posts[i], ator, "👍"); err != nil {
			return err
		}
		if _, err = tx.Exec("insert into reacoes_comentarios (comentario_id, usuario_id, emoji) values (?, ?, ?)", comentario, usuarios[i], "❤️"); err != nil {
			return err
		}
		if _, err = tx.Exec("insert into salvos (usuario_id, publicacao_id) values (?, ?)", ator, posts[i]); err != nil {
			return err
		}
		if _, err = tx.Exec("insert into notificacoes (usuario_id, ator_id, tipo, publicacao_id, comentario_id) values (?, ?, ?, ?, ?)", usuarios[i], ator, modelos.NotificacaoComentario, posts[i], comentario); err != nil {
			return err
		}
	}

	felasPost, err := inserir(tx, "insert into publicacoes (titulo, conteudo, autor_id) values (?, ?, ?)", "Para meus Felas", "Passando para agradecer por estarem por perto. Vocês deixam meus dias mais leves!", usuarios[0])
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into publicacoes_felas (publicacao_id) values (?)", felasPost); err != nil {
		return err
	}
	enquete, err := inserir(tx, "insert into publicacoes (titulo, conteudo, autor_id) values (?, ?, ?)", "Qual é o plano?", "Como vocês preferem passar uma tarde livre?", usuarios[1])
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into enquetes (publicacao_id, titulo) values (?, ?)", enquete, "Escolha seu favorito"); err != nil {
		return err
	}
	opcao, err := inserir(tx, "insert into opcoes_enquetes (publicacao_id, texto) values (?, ?)", enquete, "Passeio ao ar livre")
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into opcoes_enquetes (publicacao_id, texto) values (?, ?)", enquete, "Filme em casa"); err != nil {
		return err
	}
	if _, err = tx.Exec("insert into votos_enquetes (publicacao_id, usuario_id, opcao_id) values (?, ?, ?)", enquete, usuarios[0], opcao); err != nil {
		return err
	}
	repost, err := inserir(tx, "insert into publicacoes (titulo, conteudo, autor_id) values (?, ?, ?)", "", "Café e boas ideias combinam demais!", usuarios[1])
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into reposts (publicacao_id, original_id) values (?, ?)", repost, posts[0]); err != nil {
		return err
	}
	pasta, err := inserir(tx, "insert into pastas_salvos (usuario_id, nome) values (?, ?)", usuarios[1], "Para guardar")
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into pastas_publicacoes (pasta_id, usuario_id, publicacao_id) values (?, ?, ?)", pasta, usuarios[1], posts[0]); err != nil {
		return err
	}
	if err = conversa(tx, usuarios[0], usuarios[1]); err != nil {
		return err
	}
	if _, err = tx.Exec("insert into mensagens (conversa_id, remetente_id, texto) select id, ?, ? from conversas where usuario_menor = least(?, ?) and usuario_maior = greatest(?, ?)", usuarios[1], "Oi! Tudo bem sim. Vamos marcar aquele café esta semana?", usuarios[0], usuarios[1], usuarios[0], usuarios[1]); err != nil {
		return err
	}
	grupo, err := inserir(tx, "insert into grupos (nome, criador_id) values (?, ?)", "Papo de fim de tarde", usuarios[1])
	if err != nil {
		return err
	}
	for _, membro := range append([]int64{usuarios[1], usuarios[0]}, existentes...) {
		if _, err = tx.Exec("insert into grupo_membros (grupo_id, usuario_id, pode_adicionar) values (?, ?, ?)", grupo, membro, membro == usuarios[1]); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("insert into grupo_mensagens (grupo_id, remetente_id, texto) values (?, ?, ?), (?, ?, ?)", grupo, usuarios[1], "Oi, pessoal! O que vocês acham de marcar um café no fim de semana?", grupo, usuarios[0], "Topo! Sábado à tarde funciona para vocês?"); err != nil {
		return err
	}
	if _, err = tx.Exec("insert into silenciados (silenciador_id, silenciado_id) values (?, ?)", usuarios[6], usuarios[8]); err != nil {
		return err
	}
	if _, err = tx.Exec("insert into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", usuarios[7], usuarios[9]); err != nil {
		return err
	}
	if _, err = tx.Exec("insert into denuncias (denunciante_id, dono_id, tipo, alvo_id, motivo) values (?, ?, ?, ?, ?)", usuarios[6], usuarios[8], "perfil", usuarios[8], "spam"); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("confirmar dados de exemplo: %w", err)
	}
	return nil
}

func inserir(tx *sql.Tx, consulta string, args ...any) (int64, error) {
	resultado, err := tx.Exec(consulta, args...)
	if err != nil {
		return 0, err
	}
	return resultado.LastInsertId()
}

func seguirMutuamente(tx *sql.Tx, a, b int64) error {
	_, err := tx.Exec("insert into seguidores (usuario_id, seguidor_id) values (?, ?), (?, ?)", a, b, b, a)
	return err
}

func conversa(tx *sql.Tx, remetente, destinatario int64) error {
	menor, maior := remetente, destinatario
	if menor > maior {
		menor, maior = maior, menor
	}
	id, err := inserir(tx, "insert into conversas (usuario_menor, usuario_maior, iniciada_por, aceita) values (?, ?, ?, true)", menor, maior, remetente)
	if err != nil {
		return err
	}
	mensagem, err := inserir(tx, "insert into mensagens (conversa_id, remetente_id, texto) values (?, ?, ?)", id, remetente, "Oi! Como você está? Faz tempo que não conversamos.")
	if err != nil {
		return err
	}
	_, err = tx.Exec("insert into reacoes_mensagens (mensagem_id, usuario_id, emoji) values (?, ?, ?)", mensagem, remetente, "👍")
	return err
}
