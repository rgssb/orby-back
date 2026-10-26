CREATE DATABASE IF NOT EXISTS devebook;
USE devebook;

DROP TABLE IF EXISTS notificacoes;
DROP TABLE IF EXISTS redefinicoes_senha;
DROP TABLE IF EXISTS curtidas_comentarios;
DROP TABLE IF EXISTS comentarios;
DROP TABLE IF EXISTS mensagens;
DROP TABLE IF EXISTS bloqueios;
DROP TABLE IF EXISTS conversas;
DROP TABLE IF EXISTS banners_usuarios;
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
    titulo varchar(150) not null,
    conteudo varchar(5000) not null,

    autor_id int not null,
    FOREIGN KEY (autor_id)
    REFERENCES usuarios(id)
    ON DELETE CASCADE,

    curtidas int default 0,
    criadoEm timestamp default current_timestamp(),
    editadoEm timestamp null default null
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
    dados mediumblob not null,
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
    FOREIGN KEY (publicacao_id)
    REFERENCES publicacoes(id)
    ON DELETE CASCADE,
    FOREIGN KEY (autor_id)
    REFERENCES usuarios(id)
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
    dados mediumblob not null,
    FOREIGN KEY (publicacao_id) REFERENCES publicacoes(id) ON DELETE CASCADE
)ENGINE=INNODB;

CREATE TABLE banners_usuarios(
    usuario_id int not null primary key,
    preset varchar(20) null,
    tipo varchar(30) null,
    dados mediumblob null,
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
