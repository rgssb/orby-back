package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
)

// ErroComentarioPaiInvalido indica que a resposta aponta para um comentário que não existe ou é de outra publicação
var ErroComentarioPaiInvalido = errors.New("O comentário que você quer responder não existe")

// Comentarios representa um repositorio de comentarios
type Comentarios struct {
	db *sql.DB
}

// NovoRepositorioDeComentarios cria um repositorio de comentarios
func NovoRepositorioDeComentarios(db *sql.DB) *Comentarios {
	return &Comentarios{db: db}
}

// ResolverPai devolve o comentário principal ao qual uma resposta deve ser ligada. As conversas têm um
// só nível: responder a uma resposta liga o novo comentário ao mesmo comentário principal.
func (repositorio Comentarios) ResolverPai(publicacaoID, comentarioID uint64) (principalID, autorPrincipal uint64, erro error) {
	erro = repositorio.db.QueryRow(`
		select p.id, p.autor_id from comentarios c
		inner join comentarios p on p.id = coalesce(c.parent_id, c.id)
		where c.id = ? and c.publicacao_id = ?`, comentarioID, publicacaoID,
	).Scan(&principalID, &autorPrincipal)
	if errors.Is(erro, sql.ErrNoRows) {
		return 0, 0, ErroComentarioPaiInvalido
	}
	return
}

// Criar insere um comentario (ou uma resposta, quando ParentID esta preenchido) e retorna o seu ID
func (repositorio Comentarios) Criar(comentario modelos.Comentario) (uint64, error) {
	var parentID interface{}
	if comentario.ParentID != 0 {
		parentID = comentario.ParentID
	}
	resultado, erro := repositorio.db.Exec(
		"insert into comentarios (publicacao_id, autor_id, conteudo, parent_id) values (?, ?, ?, ?)",
		comentario.PublicacaoID, comentario.AutorID, comentario.Conteudo, parentID,
	)
	if erro != nil {
		return 0, erro
	}

	ultimoID, erro := resultado.LastInsertId()
	if erro != nil {
		return 0, erro
	}

	return uint64(ultimoID), nil
}

// BuscarPorID traz um comentario ja com o dono da publicacao (usado nas permissoes)
func (repositorio Comentarios) BuscarPorID(comentarioID uint64) (modelos.Comentario, uint64, error) {
	var (
		comentario      modelos.Comentario
		autorPublicacao uint64
	)

	erro := repositorio.db.QueryRow(`
		select c.id, c.publicacao_id, c.autor_id, c.conteudo, c.criadoEm, p.autor_id
		from comentarios c inner join publicacoes p on p.id = c.publicacao_id
		where c.id = ?`, comentarioID,
	).Scan(
		&comentario.ID,
		&comentario.PublicacaoID,
		&comentario.AutorID,
		&comentario.Conteudo,
		&comentario.CriadoEm,
		&autorPublicacao,
	)
	if erro != nil {
		return modelos.Comentario{}, 0, erro
	}

	return comentario, autorPublicacao, nil
}

// BuscarPorPublicacao lista os comentarios principais de uma publicacao, do mais antigo ao mais novo
func (repositorio Comentarios) BuscarPorPublicacao(publicacaoID, usuarioID, limite, deslocamento uint64) ([]modelos.Comentario, error) {
	return repositorio.listar("c.publicacao_id = ? and c.parent_id is null", publicacaoID, usuarioID, limite, deslocamento)
}

// BuscarRespostas lista as respostas de um comentario, da mais antiga a mais nova
func (repositorio Comentarios) BuscarRespostas(comentarioID, usuarioID, limite, deslocamento uint64) ([]modelos.Comentario, error) {
	return repositorio.listar("c.parent_id = ?", comentarioID, usuarioID, limite, deslocamento)
}

func (repositorio Comentarios) listar(filtro string, referenciaID, usuarioID, limite, deslocamento uint64) ([]modelos.Comentario, error) {
	consulta := `
		select c.id, c.publicacao_id, c.autor_id, u.nick, ` + colunaFotoVersao + `, c.conteudo, c.criadoEm,
			c.editadoEm is not null,
			(select count(*) from curtidas_comentarios cc where cc.comentario_id = c.id),
			exists(select 1 from curtidas_comentarios cc where cc.comentario_id = c.id and cc.usuario_id = ?),
			coalesce(c.parent_id, 0),
			(select count(*) from comentarios r where r.parent_id = c.id)
		from comentarios c
		inner join usuarios u on u.id = c.autor_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where ` + filtro + `
		order by c.id asc`
	argumentos := []interface{}{usuarioID, referenciaID}
	if limite > 0 {
		consulta += " limit ? offset ?"
		argumentos = append(argumentos, limite, deslocamento)
	}

	linhas, erro := repositorio.db.Query(consulta, argumentos...)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	comentarios := []modelos.Comentario{}
	for linhas.Next() {
		var comentario modelos.Comentario
		if erro = linhas.Scan(
			&comentario.ID,
			&comentario.PublicacaoID,
			&comentario.AutorID,
			&comentario.AutorNick,
			&comentario.AutorFotoVersao,
			&comentario.Conteudo,
			&comentario.CriadoEm,
			&comentario.Editado,
			&comentario.Curtidas,
			&comentario.CurtidoPorUsuario,
			&comentario.ParentID,
			&comentario.Respostas,
		); erro != nil {
			return nil, erro
		}
		comentarios = append(comentarios, comentario)
	}

	return comentarios, linhas.Err()
}

// Atualizar troca o texto de um comentario e marca como editado
func (repositorio Comentarios) Atualizar(comentarioID uint64, conteudo string) error {
	_, erro := repositorio.db.Exec("update comentarios set conteudo = ?, editadoEm = current_timestamp() where id = ?", conteudo, comentarioID)
	return erro
}

// Curtir registra a curtida de um usuario (repetir a curtida nao tem efeito)
func (repositorio Comentarios) Curtir(comentarioID, usuarioID uint64) error {
	_, erro := repositorio.db.Exec("insert ignore into curtidas_comentarios (comentario_id, usuario_id) values (?, ?)", comentarioID, usuarioID)
	return erro
}

// Descurtir remove a curtida de um usuario
func (repositorio Comentarios) Descurtir(comentarioID, usuarioID uint64) error {
	_, erro := repositorio.db.Exec("delete from curtidas_comentarios where comentario_id = ? and usuario_id = ?", comentarioID, usuarioID)
	return erro
}

// Deletar exclui um comentario
func (repositorio Comentarios) Deletar(comentarioID uint64) error {
	_, erro := repositorio.db.Exec("delete from comentarios where id = ?", comentarioID)
	return erro
}
