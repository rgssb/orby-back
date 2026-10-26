CREATE TABLE IF NOT EXISTS curtidas_publicacoes(
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
