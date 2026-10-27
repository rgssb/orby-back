package repositorios

import (
	"database/sql"
	"fmt"
	"orby/api/src/midia"
	"orby/api/src/modelos"
)

// usuarios representa um repositorio de usuarios
type usuarios struct {
	db *sql.DB
}

// colunaFotoVersao devolve o instante da última troca de foto (0 quando não há foto)
const colunaFotoVersao = "cast(coalesce(unix_timestamp(f.atualizadoEm), 0) as unsigned)"

// NovoRepositorioDeUsuarios cria um repositorio de usuarios
func NovoRepositorioDeUsuarios(db *sql.DB) *usuarios {
	return &usuarios{db}
}

// Criar insere um usuario no banco de dados
func (repositorio usuarios) Criar(usuario modelos.Usuario) (uint64, error) {
	statement, erro := repositorio.db.Prepare(
		"insert into usuarios (nome, nick, email, senha) values(?, ?, ?, ?)",
	)
	if erro != nil {
		return 0, erro
	}

	defer statement.Close()

	resultado, erro := statement.Exec(
		usuario.Nome, usuario.Nick, usuario.Email, usuario.Senha,
	)

	if erro != nil {
		return 0, erro
	}

	ultimoIDInseriodo, erro := resultado.LastInsertId()

	if erro != nil {
		return 0, erro
	}

	return uint64(ultimoIDInseriodo), nil
}

// buscar traz todos ou usuarios que atendem um filtro de nome ou nick
func (repositorio usuarios) Buscar(nomeOuNick string) ([]modelos.Usuario, error) {
	nomeOuNick = fmt.Sprintf("%%%s%%", nomeOuNick)

	linhas, erro := repositorio.db.Query(
		`select u.id, u.nome, u.nick, u.bio, u.criadoEm, `+colunaFotoVersao+`
		from usuarios u left join fotos_usuarios f on f.usuario_id = u.id
		where u.nome LIKE ? or u.nick LIKE ?`,
		nomeOuNick, nomeOuNick,
	)
	if erro != nil {
		return nil, erro
	}

	defer linhas.Close()

	var usuarios []modelos.Usuario
	for linhas.Next() {
		var usuario modelos.Usuario

		if erro = linhas.Scan(
			&usuario.ID,
			&usuario.Nome,
			&usuario.Nick,
			&usuario.Bio,
			&usuario.CriadoEm,
			&usuario.FotoVersao,
		); erro != nil {
			return nil, erro
		}

		usuarios = append(usuarios, usuario)
	}
	if erro = linhas.Err(); erro != nil {
		return nil, erro
	}
	return usuarios, nil
}

// BuscarPorID traz um usuario do Banco de dados
func (repositorio usuarios) BuscarPorID(ID uint64) (modelos.Usuario, error) {
	linhas, erro := repositorio.db.Query(
		`select u.id, u.nome, u.nick, u.email, u.bio, u.criadoEm, u.privado, `+colunaFotoVersao+`,
		coalesce(b.preset, ''), cast(if(b.tipo is null, 0, unix_timestamp(b.atualizadoEm)) as unsigned)
		from usuarios u left join fotos_usuarios f on f.usuario_id = u.id
		left join banners_usuarios b on b.usuario_id = u.id
		where u.id = ?`,
		ID,
	)
	if erro != nil {
		return modelos.Usuario{}, erro
	}
	defer linhas.Close()

	var usuario modelos.Usuario

	if linhas.Next() {
		if erro = linhas.Scan(
			&usuario.ID,
			&usuario.Nome,
			&usuario.Nick,
			&usuario.Email,
			&usuario.Bio,
			&usuario.CriadoEm,
			&usuario.Privado,
			&usuario.FotoVersao,
			&usuario.BannerPreset,
			&usuario.BannerVersao,
		); erro != nil {
			return modelos.Usuario{}, erro
		}
	}
	return usuario, nil
}

// Atualizar altera as informações no banco de dados
func (repositorio usuarios) Atualizar(ID uint64, usuario modelos.Usuario) error {
	statement, erro := repositorio.db.Prepare(
		"update usuarios set nome = ?, nick = ?, email = ?, bio = ? where id = ?",
	)
	if erro != nil {
		return erro
	}

	defer statement.Close()

	if _, erro = statement.Exec(usuario.Nome, usuario.Nick, usuario.Email, usuario.Bio, ID); erro != nil {
		return erro
	}

	return nil
}

// Deletar deleta um usuario do banco de dados
func (repositorio usuarios) Deletar(ID uint64) error {
	statement, erro := repositorio.db.Prepare("delete from usuarios where id = ?")
	if erro != nil {
		return erro
	}
	defer statement.Close()

	if _, erro = statement.Exec(ID); erro != nil {
		return erro
	}

	midia.Remover(midia.Fotos, ID)
	return midia.Remover(midia.Banners, ID)
}

// BuscarPorEmail busca um usuario por email e retorna seu id e senha com hash
func (repositorio usuarios) BuscarPorEmail(email string) (modelos.Usuario, error) {
	linha, erro := repositorio.db.Query("select id, senha from usuarios where email = ?", email)
	if erro != nil {
		return modelos.Usuario{}, erro
	}
	defer linha.Close()

	var usuario modelos.Usuario

	if linha.Next() {
		if erro = linha.Scan(&usuario.ID, &usuario.Senha); erro != nil {
			return modelos.Usuario{}, erro
		}
	}

	return usuario, nil
}

// Seguir permite que um usuario siga outro
func (repositorio usuarios) Seguir(usuarioID, seguidorID uint64) error {
	statement, erro := repositorio.db.Prepare(
		"insert ignore into seguidores (usuario_id, seguidor_id) values (?, ?)",
	)
	if erro != nil {
		return erro
	}
	defer statement.Close()

	if _, erro = statement.Exec(usuarioID, seguidorID); erro != nil {
		return erro
	}

	return nil
}

// PararDeSeguir permite que um usuario pare de seguir outro
func (repositorio usuarios) PararDeSeguir(usuarioID, seguidorID uint64) error {
	statement, erro := repositorio.db.Prepare(
		"delete from seguidores where usuario_id = ? and seguidor_id = ?",
	)
	if erro != nil {
		return erro
	}
	defer statement.Close()

	// Também cancela um pedido de seguimento pendente
	if erro = repositorio.CancelarPedido(usuarioID, seguidorID); erro != nil {
		return erro
	}

	if _, erro = statement.Exec(usuarioID, seguidorID); erro != nil {
		return erro
	}

	return nil
}

// BuscarSeguidores traz todos os seguidores de um usuario
func (repositorio usuarios) BuscarSeguidores(usuarioID uint64) ([]modelos.Usuario, error) {
	linhas, erro := repositorio.db.Query(`
		select u.id, u.nome, u.nick, u.criadoEm, `+colunaFotoVersao+`
		from usuarios u inner join seguidores s on u.id = s.seguidor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where s.usuario_id = ?`,
		usuarioID,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	var usuarios []modelos.Usuario
	for linhas.Next() {
		var usuario modelos.Usuario
		if erro = linhas.Scan(
			&usuario.ID,
			&usuario.Nome,
			&usuario.Nick,
			&usuario.CriadoEm,
			&usuario.FotoVersao,
		); erro != nil {
			return nil, erro
		}
		usuarios = append(usuarios, usuario)
	}

	return usuarios, nil
}

// BuscarSeguindo traz todos os seguidores que o usuario esta seguindo
func (repositorio usuarios) BuscarSeguindo(usuarioID uint64) ([]modelos.Usuario, error) {
	linhas, erro := repositorio.db.Query(`
	select u.id, u.nome, u.nick, u.criadoEm, `+colunaFotoVersao+`
	from usuarios u inner join seguidores s on u.id = s.usuario_id
	left join fotos_usuarios f on f.usuario_id = u.id
	where s.seguidor_id = ?`,
		usuarioID,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	var usuarios []modelos.Usuario

	for linhas.Next() {
		var usuario modelos.Usuario

		if erro = linhas.Scan(
			&usuario.ID,
			&usuario.Nome,
			&usuario.Nick,
			&usuario.CriadoEm,
			&usuario.FotoVersao,
		); erro != nil {
			return nil, erro
		}

		usuarios = append(usuarios, usuario)
	}

	return usuarios, nil

}

// Buscar senha busca a senha dentro do banco de dados pelo ID
func (repositorio usuarios) BuscarSenha(usuarioID uint64) (string, error) {
	linha, erro := repositorio.db.Query("select senha from usuarios where id = ?", usuarioID)
	if erro != nil {
		return "", erro
	}
	defer linha.Close()

	var usuario modelos.Usuario

	if linha.Next() {
		if erro = linha.Scan(&usuario.Senha); erro != nil {
			return "", erro
		}
	}

	return usuario.Senha, nil

}

// AtualizarSenha altera a senha de um usuario
func (repositorio usuarios) AtualizarSenha(usuarioID uint64, senha string) error {
	statement, erro := repositorio.db.Prepare("update usuarios set senha = ? where id = ?")
	if erro != nil {
		return erro
	}
	defer statement.Close()

	if _, erro = statement.Exec(senha, usuarioID); erro != nil {
		return erro
	}

	return nil

}

// BuscarIDPorNick encontra o usuario de um @nick, ignorando espacos e maiusculas
func (repositorio usuarios) BuscarIDPorNick(nick string) (uint64, error) {
	var id uint64
	erro := repositorio.db.QueryRow(
		"select id from usuarios where replace(nick, ' ', '') = ? order by nick = ? desc limit 1",
		nick, nick,
	).Scan(&id)
	return id, erro
}

// BuscarRelacionados traz quem o usuario segue ou quem o segue, cujo nick (sem espacos) comeca com o prefixo
func (repositorio usuarios) BuscarRelacionados(usuarioID uint64, prefixo string, limite int) ([]modelos.Sugestao, error) {
	linhas, erro := repositorio.db.Query(`
		select u.id, u.nick, `+colunaFotoVersao+`
		from usuarios u
		left join fotos_usuarios f on f.usuario_id = u.id
		where u.id <> ?
		and (
			u.id in (select usuario_id from seguidores where seguidor_id = ?)
			or u.id in (select seguidor_id from seguidores where usuario_id = ?)
		)
		and replace(u.nick, ' ', '') like ?
		order by u.nick
		limit ?`, usuarioID, usuarioID, usuarioID, prefixo+"%", limite,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	sugestoes := []modelos.Sugestao{}
	for linhas.Next() {
		var s modelos.Sugestao
		if erro = linhas.Scan(&s.ID, &s.Nick, &s.FotoVersao); erro != nil {
			return nil, erro
		}
		sugestoes = append(sugestoes, s)
	}

	return sugestoes, linhas.Err()
}
