package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
	"orby/api/src/suspensoes"
	"strings"
)

var (
	ErrUsuarioNaoEncontrado = errors.New("Usuário não encontrado")
	ErrNaoSuspendeAdmin     = errors.New("Administradores não podem ser suspensos")
)

// EhAdmin diz se a pessoa é administradora
func EhAdmin(db *sql.DB, usuarioID uint64) (admin bool, erro error) {
	erro = db.QueryRow("select admin = 1 from usuarios where id = ?", usuarioID).Scan(&admin)
	if errors.Is(erro, sql.ErrNoRows) {
		return false, nil
	}

	return
}

// Administracao representa as ações exclusivas de administradores
type Administracao struct {
	db *sql.DB
}

// NovoRepositorioDeAdministracao cria um repositorio de administracao
func NovoRepositorioDeAdministracao(db *sql.DB) *Administracao {
	return &Administracao{db: db}
}

// Suspender impede a conta de entrar e de usar o Orby até ser reativada; o conteúdo dela é mantido
func (repositorio Administracao) Suspender(usuarioID uint64) error {
	var admin bool
	if erro := repositorio.db.QueryRow("select admin = 1 from usuarios where id = ?", usuarioID).Scan(&admin); erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			return ErrUsuarioNaoEncontrado
		}
		return erro
	}
	if admin {
		return ErrNaoSuspendeAdmin
	}
	if _, erro := repositorio.db.Exec("update usuarios set suspenso_em = coalesce(suspenso_em, now()) where id = ?", usuarioID); erro != nil {
		return erro
	}
	suspensoes.Definir(usuarioID, true)
	return nil
}

// Reativar devolve o acesso à conta
func (repositorio Administracao) Reativar(usuarioID uint64) error {
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	var suspensa bool
	if erro = tx.QueryRow("select suspenso_em is not null from usuarios where id = ? for update", usuarioID).Scan(&suspensa); erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			return ErrUsuarioNaoEncontrado
		}
		return erro
	}
	if suspensa {
		if erro = resolverDenuncias(tx, usuarioID); erro != nil {
			return erro
		}
	}
	if _, erro = tx.Exec("update usuarios set suspenso_em = null where id = ?", usuarioID); erro != nil {
		return erro
	}
	if erro = tx.Commit(); erro != nil {
		return erro
	}
	suspensoes.Definir(usuarioID, false)
	return nil
}

// ListarSuspensos devolve as contas suspensas, as mais recentes primeiro
func (repositorio Administracao) ListarSuspensos() ([]modelos.ContaSuspensa, error) {
	linhas, erro := repositorio.db.Query("select id, nome, nick, suspenso_em from usuarios where suspenso_em is not null order by suspenso_em desc")
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	contas := []modelos.ContaSuspensa{}
	for linhas.Next() {
		var c modelos.ContaSuspensa
		if erro = linhas.Scan(&c.UsuarioID, &c.Nome, &c.Nick, &c.SuspensaEm); erro != nil {
			return nil, erro
		}
		contas = append(contas, c)
	}
	return contas, linhas.Err()
}

// Estatisticas resume o tamanho do Orby para o painel da administração
func (repositorio Administracao) Estatisticas() (e modelos.EstatisticasDoOrby, erro error) {
	erro = repositorio.db.QueryRow(`select
		(select count(*) from usuarios),
		(select count(*) from publicacoes),
		(select count(*) from comentarios),
		(select count(*) from usuarios where suspenso_em is not null),
		(select count(distinct dono_id) from denuncias where resolvida = 0),
		(select count(*) from usuarios where criadoEm > now() - interval 7 day)`,
	).Scan(&e.Usuarios, &e.Publicacoes, &e.Comentarios, &e.Suspensos, &e.PerfisDenunciados, &e.NovosNaSemana)
	return
}

// Listar retorna uma página de contas ou conteúdo para a administração.
func (repositorio Administracao) Listar(tipo, busca string, deslocamento uint64) ([]modelos.ItemAdministrativo, error) {
	var consulta string
	switch tipo {
	case "contas", "novas", "suspensas":
		filtro := ""
		if tipo == "novas" {
			filtro = " and u.criadoEm > now() - interval 7 day"
		} else if tipo == "suspensas" {
			filtro = " and u.suspenso_em is not null"
		}
		data := "u.criadoEm"
		if tipo == "suspensas" {
			data = "u.suspenso_em"
		}
		consulta = `select u.id, u.id, u.nome, u.nick, '', '', ` + data + `, 0, 0, u.suspenso_em is not null
			from usuarios u where (u.nome like ? or u.nick like ?)` + filtro + ` order by ` + data + ` desc, u.id desc limit 16 offset ?`
	case "publicacoes":
		consulta = `select p.id, u.id, u.nome, u.nick, p.titulo, p.conteudo, p.criadoEm, 0, if(u.privado = 1, u.id, 0), u.suspenso_em is not null
			from publicacoes p join usuarios u on u.id = p.autor_id
			where (u.nome like ? or u.nick like ? or p.titulo like ? or p.conteudo like ?)
			order by p.id desc limit 16 offset ?`
	case "comentarios":
		consulta = `select c.publicacao_id, u.id, u.nome, u.nick, '', c.conteudo, c.criadoEm, 0, if(dono.privado = 1, dono.id, 0), u.suspenso_em is not null
			from comentarios c join usuarios u on u.id = c.autor_id
			join publicacoes p on p.id = c.publicacao_id join usuarios dono on dono.id = p.autor_id
			where (u.nome like ? or u.nick like ? or c.conteudo like ?)
			order by c.id desc limit 16 offset ?`
	case "denunciadas":
		consulta = `select u.id, u.id, u.nome, u.nick, '', '', max(d.criadoEm), count(distinct d.denunciante_id), 0, u.suspenso_em is not null
			from denuncias d join usuarios u on u.id = d.dono_id
			where d.resolvida = 0 and (u.nome like ? or u.nick like ?)
			group by u.id, u.nome, u.nick, u.suspenso_em order by max(d.criadoEm) desc, u.id desc limit 16 offset ?`
	default:
		return nil, errors.New("Listagem inválida")
	}
	termo := "%" + strings.TrimSpace(busca) + "%"
	args := []interface{}{termo, termo}
	if tipo == "publicacoes" {
		args = append(args, termo, termo)
	} else if tipo == "comentarios" {
		args = append(args, termo)
	}
	args = append(args, deslocamento)
	linhas, erro := repositorio.db.Query(consulta, args...)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	itens := []modelos.ItemAdministrativo{}
	for linhas.Next() {
		var item modelos.ItemAdministrativo
		if erro = linhas.Scan(&item.ID, &item.UsuarioID, &item.Nome, &item.Nick, &item.Titulo, &item.Conteudo, &item.Data, &item.Total, &item.AlvoPrivadoID, &item.Suspenso); erro != nil {
			return nil, erro
		}
		itens = append(itens, item)
	}
	return itens, linhas.Err()
}
