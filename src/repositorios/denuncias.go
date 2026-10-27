package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
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

// Registrar guarda a denúncia (repetir a mesma denúncia não conta de novo) e avisa os administradores
// quando o dono do conteúdo chega aos limites de alerta e de revisão
func (repositorio Denuncias) Registrar(denuncianteID uint64, tipo string, alvoID uint64, motivo string) error {
	if !motivosDeDenuncia[motivo] {
		return ErrDenunciaInvalida
	}
	var consulta string
	switch tipo {
	case modelos.DenunciaPerfil:
		consulta = "select id from usuarios where id = ?"
	case modelos.DenunciaPublicacao:
		consulta = "select autor_id from publicacoes where id = ?"
	case modelos.DenunciaComentario:
		consulta = "select autor_id from comentarios where id = ?"
	default:
		return ErrDenunciaInvalida
	}

	var donoID uint64
	if erro := repositorio.db.QueryRow(consulta, alvoID).Scan(&donoID); erro != nil {
		if errors.Is(erro, sql.ErrNoRows) {
			return ErrDenunciaNaoAchada
		}
		return erro
	}
	if donoID == denuncianteID {
		return ErrDenunciaPropria
	}

	var hoje int
	if erro := repositorio.db.QueryRow(
		"select count(*) from denuncias where denunciante_id = ? and criadoEm > now() - interval 1 day", denuncianteID,
	).Scan(&hoje); erro != nil {
		return erro
	}
	if hoje >= LimiteDiarioDeDenuncias {
		return ErrDenunciasDemais
	}

	if _, erro := repositorio.db.Exec(
		"insert ignore into denuncias (denunciante_id, dono_id, tipo, alvo_id, motivo) values (?, ?, ?, ?, ?)",
		denuncianteID, donoID, tipo, alvoID, motivo,
	); erro != nil {
		return erro
	}

	var total uint64
	if erro := repositorio.db.QueryRow(
		"select count(distinct denunciante_id) from denuncias where dono_id = ? and resolvida = 0", donoID,
	).Scan(&total); erro != nil {
		return erro
	}
	if total >= DenunciasParaAlerta {
		if erro := repositorio.avisarAdmins(donoID, modelos.NotificacaoPerfilEmAlerta); erro != nil {
			return erro
		}
	}
	if total >= DenunciasParaRevisao {
		return repositorio.avisarAdmins(donoID, modelos.NotificacaoPerfilEmRevisao)
	}
	return nil
}

func (repositorio Denuncias) avisarAdmins(donoID uint64, tipo string) error {
	_, erro := repositorio.db.Exec(`
		insert into notificacoes (usuario_id, ator_id, tipo)
		select u.id, ?, ? from usuarios u
		where u.admin = 1 and u.id <> ?
		and not exists (
			select 1 from notificacoes n where n.usuario_id = u.id and n.ator_id = ? and n.tipo = ?
		)`, donoID, tipo, donoID, donoID, tipo)
	return erro
}

// ListarPerfis devolve os perfis com denúncias abertas a partir do limite de alerta, os mais denunciados primeiro
func (repositorio Denuncias) ListarPerfis() ([]modelos.PerfilDenunciado, error) {
	linhas, erro := repositorio.db.Query(`
		select u.id, u.nick, u.nome, count(distinct d.denunciante_id) as total,
			coalesce(sum(d.tipo = 'perfil'), 0), coalesce(sum(d.tipo = 'publicacao'), 0), coalesce(sum(d.tipo = 'comentario'), 0),
			max(d.criadoEm)
		from denuncias d inner join usuarios u on u.id = d.dono_id
		where d.resolvida = 0
		group by u.id, u.nick, u.nome
		having total >= ?
		order by total desc, max(d.criadoEm) desc`, DenunciasParaAlerta)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	perfis := []modelos.PerfilDenunciado{}
	for linhas.Next() {
		var p modelos.PerfilDenunciado
		if erro = linhas.Scan(&p.UsuarioID, &p.Nick, &p.Nome, &p.Total, &p.Perfis, &p.Publicacoes, &p.Comentarios, &p.UltimaEm); erro != nil {
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
	if _, erro := repositorio.db.Exec("update denuncias set resolvida = 1 where dono_id = ? and resolvida = 0", donoID); erro != nil {
		return erro
	}
	_, erro := repositorio.db.Exec(
		"delete from notificacoes where ator_id = ? and tipo in (?, ?)",
		donoID, modelos.NotificacaoPerfilEmAlerta, modelos.NotificacaoPerfilEmRevisao,
	)
	return erro
}
