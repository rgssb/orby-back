package banco

import (
	"database/sql"
	"fmt"
	"orby/api/src/midia"
)

// Migrar cria as tabelas e colunas usadas pelos recursos adicionais (bio, foto,
// comentarios e recuperacao de senha). Pode ser executada varias vezes.
func Migrar() error {
	db, erro := Conectar()
	if erro != nil {
		return erro
	}
	defer db.Close()

	// Tabelas base: num banco vazio (por exemplo, no Docker) a API cria tudo sozinha
	for _, comando := range tabelasBase {
		if _, erro = db.Exec(comando); erro != nil {
			return fmt.Errorf("migracao: %w", erro)
		}
	}

	if erro = garantirColuna(db, "usuarios", "bio", "varchar(160) not null default ''"); erro != nil {
		return erro
	}

	comandos := []string{
		`CREATE TABLE IF NOT EXISTS fotos_usuarios(
			usuario_id int not null primary key,
			tipo varchar(30) not null,
			atualizadoEm timestamp default current_timestamp() on update current_timestamp(),
			FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS banners_usuarios(
			usuario_id int not null primary key,
			preset varchar(20) null,
			tipo varchar(30) null,
			atualizadoEm timestamp default current_timestamp() on update current_timestamp(),
			FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS comentarios(
			id int auto_increment primary key,
			publicacao_id int not null,
			autor_id int not null,
			conteudo varchar(500) not null,
			criadoEm timestamp default current_timestamp(),
			FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE,
			FOREIGN KEY (autor_id) REFERENCES usuarios(id) ON DELETE CASCADE,
			INDEX idx_comentarios_publicacao (publicacao_id)
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS curtidas_comentarios(
			comentario_id int not null,
			usuario_id int not null,
			PRIMARY KEY (comentario_id, usuario_id),
			FOREIGN KEY (comentario_id) REFERENCES comentarios(id) ON DELETE CASCADE,
			FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS notificacoes(
			id int auto_increment primary key,
			usuario_id int not null,
			ator_id int not null,
			tipo varchar(30) not null,
			publicacao_id int not null,
			comentario_id int null,
			lida boolean not null default false,
			criadoEm timestamp default current_timestamp(),
			FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE,
			FOREIGN KEY (ator_id) REFERENCES usuarios(id) ON DELETE CASCADE,
			FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE,
			FOREIGN KEY (comentario_id) REFERENCES comentarios(id) ON DELETE CASCADE,
			INDEX idx_notificacoes_usuario (usuario_id, id)
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS imagens_publicacoes(
			publicacao_id int not null primary key,
			tipo varchar(30) not null,
			FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS conversas(
			id int auto_increment primary key,
			usuario_menor int not null,
			usuario_maior int not null,
			iniciada_por int not null,
			aceita boolean not null default false,
			criadoEm timestamp default current_timestamp(),
			UNIQUE KEY uq_conversa (usuario_menor, usuario_maior),
			FOREIGN KEY (usuario_menor) REFERENCES usuarios(id) ON DELETE CASCADE,
			FOREIGN KEY (usuario_maior) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS mensagens(
			id int auto_increment primary key,
			conversa_id int not null,
			remetente_id int not null,
			texto varchar(1000) not null,
			lida boolean not null default false,
			criadoEm timestamp default current_timestamp(),
			INDEX idx_mensagens_conversa (conversa_id, id),
			FOREIGN KEY (conversa_id) REFERENCES conversas(id) ON DELETE CASCADE,
			FOREIGN KEY (remetente_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS bloqueios(
			bloqueador_id int not null,
			bloqueado_id int not null,
			criadoEm timestamp default current_timestamp(),
			PRIMARY KEY (bloqueador_id, bloqueado_id),
			FOREIGN KEY (bloqueador_id) REFERENCES usuarios(id) ON DELETE CASCADE,
			FOREIGN KEY (bloqueado_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS reacoes_mensagens(
			mensagem_id int not null,
			usuario_id int not null,
			emoji varchar(16) not null,
			criadoEm timestamp default current_timestamp(),
			PRIMARY KEY (mensagem_id, usuario_id),
			FOREIGN KEY (mensagem_id) REFERENCES mensagens(id) ON DELETE CASCADE,
			FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS imagens_mensagens(
			mensagem_id int not null primary key,
			tipo varchar(30) not null,
			FOREIGN KEY (mensagem_id) REFERENCES mensagens(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS salvos(
			usuario_id int not null,
			publicacao_id int not null,
			criadoEm timestamp default current_timestamp(),
			PRIMARY KEY (usuario_id, publicacao_id),
			FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE,
			FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS pedidos_seguir(
			solicitante_id int not null,
			alvo_id int not null,
			criadoEm timestamp default current_timestamp(),
			PRIMARY KEY (solicitante_id, alvo_id),
			FOREIGN KEY (solicitante_id) REFERENCES usuarios(id) ON DELETE CASCADE,
			FOREIGN KEY (alvo_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,

		`CREATE TABLE IF NOT EXISTS redefinicoes_senha(
			id int auto_increment primary key,
			usuario_id int not null,
			token_hash char(64) not null unique,
			expiraEm datetime not null,
			criadoEm timestamp default current_timestamp(),
			FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
		) ENGINE=INNODB`,
	}

	for _, comando := range comandos {
		if _, erro = db.Exec(comando); erro != nil {
			return fmt.Errorf("migracao: %w", erro)
		}
	}

	// Avisos de novo seguidor não pertencem a nenhuma publicação
	if _, erro = db.Exec("alter table notificacoes modify publicacao_id int null"); erro != nil {
		return fmt.Errorf("migracao: %w", erro)
	}

	// limpa_*: última mensagem apagada por cada pessoa (some só para ela); desculpa: 0 sem pedido, 1 aberto, 2 encerrado
	for _, coluna := range []string{"limpa_menor", "limpa_maior"} {
		if erro = garantirColuna(db, "conversas", coluna, "int not null default 0"); erro != nil {
			return erro
		}
	}
	for _, coluna := range []string{"silenciada_menor", "silenciada_maior"} {
		if erro = garantirColuna(db, "conversas", coluna, "tinyint not null default 0"); erro != nil {
			return erro
		}
	}
	if erro = garantirColuna(db, "mensagens", "responde_a", "int null default null"); erro != nil {
		return erro
	}
	if erro = garantirColuna(db, "usuarios", "privado", "tinyint not null default 0"); erro != nil {
		return erro
	}
	if erro = garantirColuna(db, "conversas", "desculpa", "tinyint not null default 0"); erro != nil {
		return erro
	}

	if erro = garantirColuna(db, "comentarios", "editadoEm", "timestamp null default null"); erro != nil {
		return erro
	}

	if erro = garantirColuna(db, "publicacoes", "editadoEm", "timestamp null default null"); erro != nil {
		return erro
	}

	return migrarMidia(db)
}

var tabelasBase = []string{
	`CREATE TABLE IF NOT EXISTS usuarios(
		ID int auto_increment primary key,
		nome varchar(50) not null,
		nick varchar(50) not null unique,
		email varchar(50) not null unique,
		senha varchar(555) not null,
		bio varchar(160) not null default '',
		privado tinyint not null default 0,
		criadoEm timestamp default current_timestamp()
	) ENGINE=INNODB`,

	`CREATE TABLE IF NOT EXISTS seguidores(
		usuario_id int not null,
		seguidor_id int not null,
		PRIMARY KEY (usuario_id, seguidor_id),
		FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE,
		FOREIGN KEY (seguidor_id) REFERENCES usuarios(id) ON DELETE CASCADE
	) ENGINE=INNODB`,

	`CREATE TABLE IF NOT EXISTS publicacoes(
		id int auto_increment primary key,
		titulo varchar(150) not null default '',
		conteudo varchar(5000) not null,
		autor_id int not null,
		curtidas int default 0,
		criadoEm timestamp default current_timestamp(),
		editadoEm timestamp null default null,
		FOREIGN KEY (autor_id) REFERENCES usuarios(id) ON DELETE CASCADE
	) ENGINE=INNODB`,

	`CREATE TABLE IF NOT EXISTS curtidas_publicacoes(
		publicacao_id int not null,
		usuario_id int not null,
		PRIMARY KEY (publicacao_id, usuario_id),
		FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE,
		FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
	) ENGINE=INNODB`,
}

// tabelasDeMidia liga cada tipo de arquivo em disco à tabela que o registra
var tabelasDeMidia = []struct{ tabela, chave, tipo string }{
	{"fotos_usuarios", "usuario_id", midia.Fotos},
	{"banners_usuarios", "usuario_id", midia.Banners},
	{"imagens_publicacoes", "publicacao_id", midia.Publicacoes},
	{"imagens_mensagens", "mensagem_id", midia.Mensagens},
}

// migrarMidia move para o disco as imagens que ainda estão em colunas "dados" (versões antigas)
// e remove a coluna. Pode ser executada várias vezes.
func migrarMidia(db *sql.DB) error {
	for _, t := range tabelasDeMidia {
		var existe int
		if erro := db.QueryRow(
			`select count(*) from information_schema.columns
			where table_schema = database() and table_name = ? and column_name = 'dados'`, t.tabela,
		).Scan(&existe); erro != nil {
			return fmt.Errorf("migracao de midia: %w", erro)
		}
		if existe == 0 {
			continue
		}

		linhas, erro := db.Query(fmt.Sprintf("select %s, dados from %s where dados is not null", t.chave, t.tabela))
		if erro != nil {
			return fmt.Errorf("migracao de midia: %w", erro)
		}
		for linhas.Next() {
			var id uint64
			var dados []byte
			if erro = linhas.Scan(&id, &dados); erro == nil {
				erro = midia.Gravar(t.tipo, id, dados)
			}
			if erro != nil {
				linhas.Close()
				return fmt.Errorf("migracao de midia (%s): %w", t.tabela, erro)
			}
		}
		if erro = linhas.Err(); erro != nil {
			return fmt.Errorf("migracao de midia: %w", erro)
		}
		linhas.Close()

		if _, erro = db.Exec(fmt.Sprintf("alter table %s drop column dados", t.tabela)); erro != nil {
			return fmt.Errorf("migracao de midia: %w", erro)
		}
	}
	return nil
}

func garantirColuna(db *sql.DB, tabela, coluna, definicao string) error {
	var existe int
	erro := db.QueryRow(
		`select count(*) from information_schema.columns
		where table_schema = database() and table_name = ? and column_name = ?`,
		tabela, coluna,
	).Scan(&existe)
	if erro != nil {
		return erro
	}

	if existe > 0 {
		return nil
	}

	_, erro = db.Exec(fmt.Sprintf("alter table %s add column %s %s", tabela, coluna, definicao))
	return erro
}
