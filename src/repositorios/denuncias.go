package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
	"strings"

	"github.com/go-sql-driver/mysql"
)

const (
	// DenunciasParaAlerta é quantas pessoas diferentes precisam denunciar para o perfil entrar em alerta
	DenunciasParaAlerta = 10
	// DenunciasParaRevisao é quantas pessoas diferentes precisam denunciar para o perfil ir à revisão da administração
	DenunciasParaRevisao = 20
	// LimiteDiarioDeDenuncias evita que uma mesma conta use a denúncia como arma
	LimiteDiarioDeDenuncias = 30
)

// Motivos aceitos numa denúncia
var motivosDeDenuncia = map[string]bool{
	"spam": true, "assedio": true, "falso": true, "improprio": true, "outro": true,
}

var (
	ErrDenunciaInvalida    = errors.New("Denúncia inválida")
	ErrDenunciaPropria     = errors.New("Você não pode denunciar a si mesmo")
	ErrDenunciaNaoAchada   = errors.New("Conteúdo não encontrado")
	ErrDenunciasDemais     = errors.New("Você já enviou denúncias demais hoje. Tente novamente amanhã")
	ErrJaDenunciado        = errors.New("Você já denunciou este conteúdo")
	ErrSemPermissaoDeAdmin = errors.New("Sem permissão")
)

// Denuncias representa um repositorio de denuncias
type Denuncias struct {
	db *sql.DB
}

// NovoRepositorioDeDenuncias cria um repositorio de denuncias
func NovoRepositorioDeDenuncias(db *sql.DB) *Denuncias {
	return &Denuncias{db: db}
}

// Registrar guarda a denúncia e avisa administradores quando necessário.
func (repositorio Denuncias) Registrar(denuncianteID uint64, tipo string, alvoID uint64, motivo string) error {
	_, erro := repositorio.RegistrarComDono(denuncianteID, tipo, alvoID, motivo)
	return erro
}

// RegistrarComDono devolve o responsável pelo conteúdo denunciado.
func (repositorio Denuncias) RegistrarComDono(denuncianteID uint64, tipo string, alvoID uint64, motivo string) (uint64, error) {
	if !motivosDeDenuncia[motivo] {
		return 0, ErrDenunciaInvalida
	}
	var consulta string
	argumentos := []interface{}{alvoID}
	switch tipo {
	case modelos.DenunciaPerfil:
		consulta = "select id from usuarios where id = ?"
	case modelos.DenunciaPublicacao:
		consulta = "select p.autor_id from publicacoes p join usuarios u on u.id = p.autor_id where p.id = ? and " + postVisivel
		argumentos = append(argumentos, denuncianteID, denuncianteID, denuncianteID, denuncianteID)
	case modelos.DenunciaComentario:
		consulta = "select c.autor_id from comentarios c join publicacoes p on p.id = c.publicacao_id join usuarios u on u.id = p.autor_id where c.id = ? and " + postVisivel
		argumentos = append(argumentos, denuncianteID, denuncianteID, denuncianteID, denuncianteID)
	default:
		return 0, ErrDenunciaInvalida
	}

	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return 0, erro
	}
	defer tx.Rollback()
	var donoID uint64
	if erro := tx.QueryRow(consulta, argumentos...).Scan(&donoID); erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			return 0, ErrDenunciaNaoAchada
		}
		return 0, erro
	}
	if donoID == denuncianteID {
		return 0, ErrDenunciaPropria
	}

	var existente bool
	if erro = tx.QueryRow("select exists(select 1 from denuncias where denunciante_id = ? and tipo = ? and alvo_id = ? and resolvida = 0)", denuncianteID, tipo, alvoID).Scan(&existente); erro != nil {
		return 0, erro
	}
	if existente {
		return 0, ErrJaDenunciado
	}
	var hoje int
	if erro := tx.QueryRow(
		"select count(*) from denuncias where denunciante_id = ? and criadoEm > now() - interval 1 day", denuncianteID,
	).Scan(&hoje); erro != nil {
		return 0, erro
	}
	if hoje >= LimiteDiarioDeDenuncias {
		return 0, ErrDenunciasDemais
	}

	if _, erro := tx.Exec(
		"insert into denuncias (denunciante_id, dono_id, tipo, alvo_id, motivo) values (?, ?, ?, ?, ?)",
		denuncianteID, donoID, tipo, alvoID, motivo,
	); erro != nil {
		var erroMySQL *mysql.MySQLError
		if errors.As(erro, &erroMySQL) && erroMySQL.Number == 1062 {
			return 0, ErrJaDenunciado
		}
		return 0, erro
	}

	var total uint64
	if erro := tx.QueryRow(
		"select count(distinct denunciante_id) from denuncias where dono_id = ? and resolvida = 0", donoID,
	).Scan(&total); erro != nil {
		return 0, erro
	}
	if total >= DenunciasParaAlerta {
		if erro := avisarAdmins(tx, donoID, modelos.NotificacaoPerfilEmAlerta); erro != nil {
			return 0, erro
		}
	}
	if total >= DenunciasParaRevisao {
		if erro := avisarAdmins(tx, donoID, modelos.NotificacaoPerfilEmRevisao); erro != nil {
			return 0, erro
		}
	}
	if erro = tx.Commit(); erro != nil {
		return 0, erro
	}
	return donoID, nil
}

func avisarAdmins(tx *sql.Tx, donoID uint64, tipo string) error {
	_, erro := tx.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo)
		select u.id, ?, ? from usuarios u
		where u.admin = 1 and u.id <> ?
		and not exists (
			select 1 from notificacoes n where n.usuario_id = u.id and n.ator_id = ? and n.tipo = ?
		)`, donoID, tipo, donoID, donoID, tipo)
	return erro
}

// Consultar devolve apenas os alvos que o usuário já denunciou.
func (repositorio Denuncias) Consultar(usuarioID uint64, alvos []modelos.AlvoDeDenuncia) ([]modelos.AlvoDeDenuncia, error) {
	if len(alvos) == 0 || len(alvos) > 100 {
		return nil, ErrDenunciaInvalida
	}
	argumentos := []interface{}{usuarioID}
	condicoes := make([]string, 0, len(alvos))
	for _, alvo := range alvos {
		if alvo.AlvoID == 0 || (alvo.Tipo != modelos.DenunciaPerfil && alvo.Tipo != modelos.DenunciaPublicacao && alvo.Tipo != modelos.DenunciaComentario) {
			return nil, ErrDenunciaInvalida
		}
		condicoes = append(condicoes, "(tipo = ? and alvo_id = ?)")
		argumentos = append(argumentos, alvo.Tipo, alvo.AlvoID)
	}
	linhas, erro := repositorio.db.Query("select tipo, alvo_id from denuncias where denunciante_id = ? and resolvida = 0 and ("+strings.Join(condicoes, " or ")+")", argumentos...)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	resultados := []modelos.AlvoDeDenuncia{}
	for linhas.Next() {
		var alvo modelos.AlvoDeDenuncia
		if erro = linhas.Scan(&alvo.Tipo, &alvo.AlvoID); erro != nil {
			return nil, erro
		}
		resultados = append(resultados, alvo)
	}
	return resultados, linhas.Err()
}

// ListarPerfis devolve os perfis com denúncias abertas a partir do limite de alerta, os mais denunciados primeiro
func (repositorio Denuncias) ListarPerfis() ([]modelos.PerfilDenunciado, error) {
	linhas, erro := repositorio.db.Query(`
		select u.id, u.nick, u.nome, count(distinct d.denunciante_id) as total,
			coalesce(sum(d.tipo = 'perfil'), 0), coalesce(sum(d.tipo = 'publicacao'), 0), coalesce(sum(d.tipo = 'comentario'), 0),
			max(d.criadoEm), u.suspenso_em is not null
		from denuncias d inner join usuarios u on u.id = d.dono_id
		where d.resolvida = 0
		group by u.id, u.nick, u.nome, u.suspenso_em
		having total >= ?
		order by total desc, max(d.criadoEm) desc`, DenunciasParaAlerta)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	perfis := []modelos.PerfilDenunciado{}
	for linhas.Next() {
		var p modelos.PerfilDenunciado
		if erro = linhas.Scan(&p.UsuarioID, &p.Nick, &p.Nome, &p.Total, &p.Perfis, &p.Publicacoes, &p.Comentarios, &p.UltimaEm, &p.Suspenso); erro != nil {
			return nil, erro
		}
		p.Nivel = "alerta"
		if p.Total >= DenunciasParaRevisao {
			p.Nivel = "revisao"
		}
		p.Motivos = map[string]int{}
		perfis = append(perfis, p)
	}
	if erro = linhas.Err(); erro != nil {
		return nil, erro
	}

	for i := range perfis {
		motivos, erro := repositorio.db.Query(
			"select motivo, count(*) from denuncias where dono_id = ? and resolvida = 0 group by motivo", perfis[i].UsuarioID)
		if erro != nil {
			return nil, erro
		}
		for motivos.Next() {
			var motivo string
			var quantidade int
			if erro = motivos.Scan(&motivo, &quantidade); erro != nil {
				motivos.Close()
				return nil, erro
			}
			perfis[i].Motivos[motivo] = quantidade
		}
		motivos.Close()
	}
	return perfis, nil
}

// Resolver arquiva as denúncias abertas contra a pessoa e limpa os avisos dos administradores
func (repositorio Denuncias) Resolver(donoID uint64) error {
	return repositorio.resolver(donoID, 0)
}

func (repositorio Denuncias) ResolverPorAdmin(donoID, adminID uint64) error {
	return repositorio.resolver(donoID, adminID)
}

func (repositorio Denuncias) resolver(donoID, adminID uint64) error {
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	if erro = resolverDenuncias(tx, donoID); erro != nil {
		return erro
	}
	if adminID != 0 {
		if erro = registrarAcaoModeracao(tx, adminID, donoID, "arquivar"); erro != nil {
			return erro
		}
	}
	return tx.Commit()
}

func resolverDenuncias(tx *sql.Tx, donoID uint64) error {
	if _, erro := tx.Exec("update denuncias set resolvida = 1 where dono_id = ? and resolvida = 0", donoID); erro != nil {
		return erro
	}
	_, erro := tx.Exec(
		"delete from notificacoes where ator_id = ? and tipo in (?, ?)",
		donoID, modelos.NotificacaoPerfilEmAlerta, modelos.NotificacaoPerfilEmRevisao,
	)
	return erro
}
