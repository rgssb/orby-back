// Package demonstracao cria dados fictícios apenas quando solicitado.
package demonstracao

import (
	"database/sql"
	"errors"
	"fmt"
	"orby/api/src/modelos"
	"orby/api/src/seguranca"
)

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
			values (?, ?, ?)`, fmt.Sprintf("Publicação de exemplo %d", i+1),
			fmt.Sprintf("Olá! Sou Usuario%d. Esta publicação é um exemplo para conhecer o Orby. #exemplo", i+1), usuarios[i])
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
		comentario, err := inserir(tx, "insert into comentarios (publicacao_id, autor_id, conteudo) values (?, ?, ?)", posts[i], ator, "Comentário de exemplo: gostei da publicação!")
		if err != nil {
			return err
		}
		if _, err = tx.Exec("insert into comentarios (publicacao_id, autor_id, conteudo, parent_id) values (?, ?, ?, ?)", posts[i], usuarios[i], "Obrigado! Esta é uma resposta de exemplo.", comentario); err != nil {
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

	felasPost, err := inserir(tx, "insert into publicacoes (titulo, conteudo, autor_id) values (?, ?, ?)", "Para meus Felas", "Esta é uma publicação de exemplo visível para meus Felas.", usuarios[0])
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into publicacoes_felas (publicacao_id) values (?)", felasPost); err != nil {
		return err
	}
	enquete, err := inserir(tx, "insert into publicacoes (titulo, conteudo, autor_id) values (?, ?, ?)", "Enquete de exemplo", "Qual recurso do Orby você mais gosta?", usuarios[1])
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into enquetes (publicacao_id, titulo) values (?, ?)", enquete, "Escolha uma opção"); err != nil {
		return err
	}
	opcao, err := inserir(tx, "insert into opcoes_enquetes (publicacao_id, texto) values (?, ?)", enquete, "Publicações")
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into opcoes_enquetes (publicacao_id, texto) values (?, ?)", enquete, "Mensagens"); err != nil {
		return err
	}
	if _, err = tx.Exec("insert into votos_enquetes (publicacao_id, usuario_id, opcao_id) values (?, ?, ?)", enquete, usuarios[0], opcao); err != nil {
		return err
	}
	repost, err := inserir(tx, "insert into publicacoes (titulo, conteudo, autor_id) values (?, ?, ?)", "", "Repost de exemplo", usuarios[1])
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into reposts (publicacao_id, original_id) values (?, ?)", repost, posts[0]); err != nil {
		return err
	}
	pasta, err := inserir(tx, "insert into pastas_salvos (usuario_id, nome) values (?, ?)", usuarios[1], "Exemplos")
	if err != nil {
		return err
	}
	if _, err = tx.Exec("insert into pastas_publicacoes (pasta_id, usuario_id, publicacao_id) values (?, ?, ?)", pasta, usuarios[1], posts[0]); err != nil {
		return err
	}
	if err = conversa(tx, usuarios[0], usuarios[1]); err != nil {
		return err
	}
	grupo, err := inserir(tx, "insert into grupos (nome, criador_id) values (?, ?)", "Grupo de exemplo", usuarios[1])
	if err != nil {
		return err
	}
	for _, membro := range append([]int64{usuarios[1], usuarios[0]}, existentes...) {
		if _, err = tx.Exec("insert into grupo_membros (grupo_id, usuario_id, pode_adicionar) values (?, ?, ?)", grupo, membro, membro == usuarios[1]); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("insert into grupo_mensagens (grupo_id, remetente_id, texto) values (?, ?, ?)", grupo, usuarios[1], "Olá! Este grupo é um exemplo para testar conversas."); err != nil {
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
	mensagem, err := inserir(tx, "insert into mensagens (conversa_id, remetente_id, texto) values (?, ?, ?)", id, remetente, "Olá! Esta é uma mensagem de exemplo.")
	if err != nil {
		return err
	}
	_, err = tx.Exec("insert into reacoes_mensagens (mensagem_id, usuario_id, emoji) values (?, ?, ?)", mensagem, remetente, "👍")
	return err
}
