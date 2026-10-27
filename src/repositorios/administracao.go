package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
	"orby/api/src/suspensoes"
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
	if _, erro := repositorio.db.Exec("update usuarios set suspenso_em = null where id = ?", usuarioID); erro != nil {
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
