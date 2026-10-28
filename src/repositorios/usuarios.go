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
		"insert into usuarios (nome, nick, email, senha, termos_aceitos_em) values(?, ?, ?, ?, now())",
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
		`select u.id, u.nome, u.nick, u.email, u.bio, u.criadoEm, u.privado, u.admin, u.suspenso_em is not null, `+colunaFotoVersao+`,
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
			&usuario.Admin,
			&usuario.Suspenso,
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

// Deletar apaga a conta e tudo que depende dela (publicações, comentários, conversas...) e, em seguida,
// os arquivos de mídia que ficaram sem registro: foto, banner, imagens das publicações e das mensagens.
func (repositorio usuarios) Deletar(ID uint64) error {
	imagensDePublicacoes, erro := repositorio.idsDeMidia(
		"select publicacao_id from imagens_publicacoes ip inner join publicacoes p on p.id = ip.publicacao_id where p.autor_id = ?", ID)
	if erro != nil {
		return erro
	}
	imagensExtras, erro := repositorio.idsDeMidia(
		"select ie.id from imagens_extras_publicacoes ie inner join publicacoes p on p.id = ie.publicacao_id where p.autor_id = ?", ID)
	if erro != nil {
		return erro
	}
	imagensDeMensagens, erro := repositorio.idsDeMidia(`
		select im.mensagem_id from imagens_mensagens im
		inner join mensagens m on m.id = im.mensagem_id
		inner join conversas c on c.id = m.conversa_id
		where c.usuario_menor = ? or c.usuario_maior = ?`, ID, ID)
	if erro != nil {
		return erro
	}

	// reposts de publicações desta conta não fazem sentido sem o original
	reposts, erro := repositorio.idsDeMidia(
		"select r.publicacao_id from reposts r inner join publicacoes o on o.id = r.original_id where o.autor_id = ?", ID)
	if erro != nil {
		return erro
	}
	for _, id := range reposts {
		if _, erro = repositorio.db.Exec("delete from publicacoes where id = ?", id); erro != nil {
			return erro
		}
	}

	// os grupos criados por esta conta passam ao membro mais antigo; os que ficariam vazios somem junto com a conta
	if _, erro = repositorio.db.Exec(`
		update grupos g set g.criador_id = (
			select m.usuario_id from grupo_membros m where m.grupo_id = g.id and m.usuario_id <> g.criador_id order by m.entrouEm, m.usuario_id limit 1)
		where g.criador_id = ? and exists (select 1 from grupo_membros m where m.grupo_id = g.id and m.usuario_id <> g.criador_id)`, ID); erro != nil {
		return erro
	}

	if _, erro = repositorio.db.Exec("delete from usuarios where id = ?", ID); erro != nil {
		return erro
	}

	// a conta já foi apagada; arquivo que não puder ser removido agora é recolhido pela limpeza periódica de órfãos
	for _, id := range imagensDePublicacoes {
		midia.Remover(midia.Publicacoes, id)
	}
	for _, id := range imagensExtras {
		midia.Remover(midia.Extras, id)
	}
	for _, id := range imagensDeMensagens {
		midia.Remover(midia.Mensagens, id)
	}
	midia.Remover(midia.Fotos, ID)
	midia.Remover(midia.Banners, ID)
	return nil
}

func (repositorio usuarios) idsDeMidia(consulta string, argumentos ...interface{}) ([]uint64, error) {
	linhas, erro := repositorio.db.Query(consulta, argumentos...)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	var ids []uint64
	for linhas.Next() {
		var id uint64
		if erro = linhas.Scan(&id); erro != nil {
			return nil, erro
		}
		ids = append(ids, id)
	}
	return ids, linhas.Err()
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
