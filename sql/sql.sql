CREATE DATABASE IF NOT EXISTS orby;
USE orby;

DROP TABLE IF EXISTS pedidos_seguir;
DROP TABLE IF EXISTS salvos;
DROP TABLE IF EXISTS grupo_mensagens;
DROP TABLE IF EXISTS grupo_membros;
DROP TABLE IF EXISTS grupos;
DROP TABLE IF EXISTS reacoes_mensagens;
DROP TABLE IF EXISTS imagens_mensagens;
DROP TABLE IF EXISTS notificacoes;
DROP TABLE IF EXISTS verificacoes_email;
DROP TABLE IF EXISTS redefinicoes_senha;
DROP TABLE IF EXISTS curtidas_comentarios;
DROP TABLE IF EXISTS comentarios;
DROP TABLE IF EXISTS mensagens;
DROP TABLE IF EXISTS reposts;
DROP TABLE IF EXISTS silenciados;
DROP TABLE IF EXISTS bloqueios;
DROP TABLE IF EXISTS conversas;
DROP TABLE IF EXISTS banners_usuarios;
DROP TABLE IF EXISTS imagens_extras_publicacoes;
DROP TABLE IF EXISTS imagens_publicacoes;
DROP TABLE IF EXISTS fotos_usuarios;
DROP TABLE IF EXISTS curtidas_publicacoes;
DROP TABLE IF EXISTS publicacoes;
DROP TABLE IF EXISTS seguidores;
DROP TABLE IF EXISTS usuarios;

CREATE TABLE usuarios(
    ID int auto_increment primary key,
    nome varchar(50) not null,
    nick varchar(50) not null unique,
    email varchar(50) not null unique,
    senha varchar(555) not null,
    bio varchar(160) not null default '',
    privado tinyint not null default 0,
    termos_aceitos_em timestamp null default null,
    email_verificado_em timestamp null default null,
    criadoEm timestamp default current_timestamp()
)ENGINE=INNODB;

CREATE TABLE seguidores(
    usuario_id int not null,
    FOREIGN KEY (usuario_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE,
    
    seguidor_id int not null,
    FOREIGN KEY (seguidor_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE,

    primary key(usuario_id, seguidor_id)
)ENGINE=INNODB;

CREATE TABLE publicacoes(
    id int auto_increment primary key,
    titulo varchar(150) not null default '',
    conteudo varchar(5000) not null,

    autor_id int not null,
    FOREIGN KEY (autor_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE,

    curtidas int default 0,
    criadoEm timestamp default current_timestamp(),
    editadoEm timestamp null default null,
    INDEX idx_publicacoes_criado (criadoEm)
)ENGINE=INNODB;

CREATE TABLE curtidas_publicacoes(
    publicacao_id int not null,
    usuario_id int not null,
    primary key(publicacao_id, usuario_id),
    FOREIGN KEY (publicacao_id)
    REFERENCES publicacoes(id)
    ON DELETE CASCADE,
    FOREIGN KEY (usuario_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE
)ENGINE=INNODB;

CREATE TABLE fotos_usuarios(
    usuario_id int not null primary key,
    tipo varchar(30) not null,
    atualizadoEm timestamp default current_timestamp() on update current_timestamp(),
    FOREIGN KEY (usuario_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE
)ENGINE=INNODB;

CREATE TABLE comentarios(
    id int auto_increment primary key,
    publicacao_id int not null,
    autor_id int not null,
    conteudo varchar(500) not null,
    criadoEm timestamp default current_timestamp(),
    editadoEm timestamp null default null,
    parent_id int null default null,
    FOREIGN KEY (publicacao_id)
    REFERENCES publicacoes(id)
    ON DELETE CASCADE,
    FOREIGN KEY (autor_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE,
    CONSTRAINT fk_comentarios_parent FOREIGN KEY (parent_id)
    REFERENCES comentarios(id)
    ON DELETE CASCADE,
    INDEX idx_comentarios_publicacao (publicacao_id)
)ENGINE=INNODB;

CREATE TABLE curtidas_comentarios(
    comentario_id int not null,
    usuario_id int not null,
    PRIMARY KEY (comentario_id, usuario_id),
    FOREIGN KEY (comentario_id) REFERENCES comentarios(id) ON DELETE CASCADE,
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
)ENGINE=INNODB;

CREATE TABLE verificacoes_email(
    id int auto_increment primary key,
    usuario_id int not null,
    token_hash char(64) not null unique,
    expiraEm datetime not null,
    criadoEm timestamp default current_timestamp(),
    FOREIGN KEY (usuario_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE
)ENGINE=INNODB;

CREATE TABLE redefinicoes_senha(
    id int auto_increment primary key,
    usuario_id int not null,
    token_hash char(64) not null unique,
    expiraEm datetime not null,
    criadoEm timestamp default current_timestamp(),
    FOREIGN KEY (usuario_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE
)ENGINE=INNODB;

CREATE TABLE notificacoes(
    id int auto_increment primary key,
    usuario_id int not null,
    ator_id int not null,
    tipo varchar(30) not null,
    publicacao_id int null,
    comentario_id int null,
    lida boolean not null default false,
    criadoEm timestamp default current_timestamp(),
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE,
    FOREIGN KEY (ator_id) REFERENCES usuarios(id) ON DELETE CASCADE,
    FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE,
    FOREIGN KEY (comentario_id) REFERENCES comentarios(id) ON DELETE CASCADE,
    INDEX idx_notificacoes_usuario (usuario_id, id)
)ENGINE=INNODB;

CREATE TABLE imagens_publicacoes(
    publicacao_id int not null primary key,
    tipo varchar(30) not null,
    FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE
)ENGINE=INNODB;

-- Da segunda imagem em diante; a primeira fica em imagens_publicacoes
CREATE TABLE imagens_extras_publicacoes(
    id int auto_increment primary key,
    publicacao_id int not null,
    tipo varchar(30) not null,
    FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE,
    INDEX idx_imagens_extras_publicacao (publicacao_id, id)
)ENGINE=INNODB;

CREATE TABLE banners_usuarios(
    usuario_id int not null primary key,
    preset varchar(20) null,
    tipo varchar(30) null,
    atualizadoEm timestamp default current_timestamp() on update current_timestamp(),
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
)ENGINE=INNODB;

CREATE TABLE conversas(
    id int auto_increment primary key,
    usuario_menor int not null,
    usuario_maior int not null,
    iniciada_por int not null,
    aceita boolean not null default false,
    limpa_menor int not null default 0,
    limpa_maior int not null default 0,
    desculpa tinyint not null default 0,
    silenciada_menor tinyint not null default 0,
    silenciada_maior tinyint not null default 0,
    criadoEm timestamp default current_timestamp(),
    UNIQUE KEY uq_conversa (usuario_menor, usuario_maior),
    FOREIGN KEY (usuario_menor) REFERENCES usuarios(id) ON DELETE CASCADE,
    FOREIGN KEY (usuario_maior) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE mensagens(
    id int auto_increment primary key,
    conversa_id int not null,
    remetente_id int not null,
    texto varchar(1000) not null,
    responde_a int null default null,
    lida boolean not null default false,
    criadoEm timestamp default current_timestamp(),
    INDEX idx_mensagens_conversa (conversa_id, id),
    FOREIGN KEY (conversa_id) REFERENCES conversas(id) ON DELETE CASCADE,
    FOREIGN KEY (remetente_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE bloqueios(
    bloqueador_id int not null,
    bloqueado_id int not null,
    criadoEm timestamp default current_timestamp(),
    PRIMARY KEY (bloqueador_id, bloqueado_id),
    FOREIGN KEY (bloqueador_id) REFERENCES usuarios(id) ON DELETE CASCADE,
    FOREIGN KEY (bloqueado_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE reposts(
    publicacao_id int not null primary key,
    original_id int not null,
    FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE,
    FOREIGN KEY (original_id) REFERENCES publicacoes(id) ON DELETE CASCADE,
    INDEX idx_reposts_original (original_id)
) ENGINE=INNODB;

CREATE TABLE silenciados(
    silenciador_id int not null,
    silenciado_id int not null,
    criadoEm timestamp default current_timestamp(),
    PRIMARY KEY (silenciador_id, silenciado_id),
    FOREIGN KEY (silenciador_id) REFERENCES usuarios(id) ON DELETE CASCADE,
    FOREIGN KEY (silenciado_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE grupos(
    id int auto_increment primary key,
    nome varchar(60) not null,
    criador_id int not null,
    criadoEm timestamp default current_timestamp(),
    FOREIGN KEY (criador_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE grupo_membros(
    grupo_id int not null,
    usuario_id int not null,
    pode_adicionar tinyint not null default 0,
    lida_ate int not null default 0,
    entrouEm timestamp default current_timestamp(),
    PRIMARY KEY (grupo_id, usuario_id),
    FOREIGN KEY (grupo_id) REFERENCES grupos(id) ON DELETE CASCADE,
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE grupo_mensagens(
    id int auto_increment primary key,
    grupo_id int not null,
    remetente_id int not null,
    texto varchar(1000) not null,
    criadoEm timestamp default current_timestamp(),
    INDEX idx_grupo_mensagens (grupo_id, id),
    FOREIGN KEY (grupo_id) REFERENCES grupos(id) ON DELETE CASCADE,
    FOREIGN KEY (remetente_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE reacoes_mensagens(
    mensagem_id int not null,
    usuario_id int not null,
    emoji varchar(16) not null,
    criadoEm timestamp default current_timestamp(),
    PRIMARY KEY (mensagem_id, usuario_id),
    FOREIGN KEY (mensagem_id) REFERENCES mensagens(id) ON DELETE CASCADE,
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE imagens_mensagens(
    mensagem_id int not null primary key,
    tipo varchar(30) not null,
    FOREIGN KEY (mensagem_id) REFERENCES mensagens(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE salvos(
    usuario_id int not null,
    publicacao_id int not null,
    criadoEm timestamp default current_timestamp(),
    PRIMARY KEY (usuario_id, publicacao_id),
    FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE,
    FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE
) ENGINE=INNODB;

CREATE TABLE pedidos_seguir(
    solicitante_id int not null,
    alvo_id int not null,
    criadoEm timestamp default current_timestamp(),
    PRIMARY KEY (solicitante_id, alvo_id),
    FOREIGN KEY (solicitante_id) REFERENCES usuarios(id) ON DELETE CASCADE,
    FOREIGN KEY (alvo_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=INNODB;
