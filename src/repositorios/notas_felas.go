package repositorios

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNotasIndisponiveis = errors.New("Notas indisponíveis")

type NotaFela struct {
	ID          uint64    `json:"id"`
	AutorID     uint64    `json:"autorId"`
	Nome        string    `json:"nome"`
	Nick        string    `json:"nick"`
	FotoVersao  uint64    `json:"fotoVersao"`
	Conteudo    string    `json:"conteudo"`
	CriadaEm    time.Time `json:"criadaEm"`
	PodeExcluir bool      `json:"podeExcluir"`
}

type NotasFelas struct {
	Ativas       bool       `json:"ativas"`
	PodeEscrever bool       `json:"podeEscrever"`
	Notas        []NotaFela `json:"notas"`
	Proxima      uint64     `json:"proxima"`
}

func ListarNotasFelas(db *sql.DB, perfil, visitante, antes uint64) (NotasFelas, error) {
	resultado := NotasFelas{Notas: []NotaFela{}}
	var acesso bool
	err := db.QueryRow(`
		select u.notas_felas_ativas,
			(u.id = ? or ((u.privado = 0 or exists(select 1 from seguidores s where s.usuario_id = u.id and s.seguidor_id = ?))
				and not exists(select 1 from bloqueios b where (b.bloqueador_id = u.id and b.bloqueado_id = ?) or (b.bloqueador_id = ? and b.bloqueado_id = u.id)))),
			(u.id <> ? and exists(select 1 from felas f where f.dono_id = u.id and f.membro_id = ?))
		from usuarios u where u.id = ? and u.suspenso_em is null`,
		visitante, visitante, visitante, visitante, visitante, visitante, perfil).Scan(&resultado.Ativas, &acesso, &resultado.PodeEscrever)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !acesso) {
		return resultado, ErrNotasIndisponiveis
	}
	if err != nil {
		return resultado, err
	}
	resultado.PodeEscrever = resultado.Ativas && resultado.PodeEscrever
	if !resultado.Ativas {
		return resultado, nil
	}
	linhas, err := db.Query(`
		select n.id, n.autor_id, a.nome, a.nick, `+colunaFotoVersao+`, n.conteudo, n.criadoEm
		from notas_felas n join usuarios a on a.id = n.autor_id
		join usuarios u on u.id = n.perfil_id
		left join fotos_usuarios f on f.usuario_id = a.id
		where n.perfil_id = ? and u.notas_felas_ativas = 1 and u.suspenso_em is null and (? = 0 or n.id < ?)
			and (u.id = ? or ((u.privado = 0 or exists(select 1 from seguidores s where s.usuario_id = u.id and s.seguidor_id = ?))
				and not exists(select 1 from bloqueios b where (b.bloqueador_id = u.id and b.bloqueado_id = ?) or (b.bloqueador_id = ? and b.bloqueado_id = u.id))))
			and (u.id = ? or (not exists(select 1 from bloqueios b where (b.bloqueador_id = a.id and b.bloqueado_id = ?) or (b.bloqueador_id = ? and b.bloqueado_id = a.id))
				and not exists(select 1 from bloqueios b where (b.bloqueador_id = a.id and b.bloqueado_id = u.id) or (b.bloqueador_id = u.id and b.bloqueado_id = a.id))))
		order by n.id desc limit 21`,
		perfil, antes, antes, visitante, visitante, visitante, visitante, visitante, visitante, visitante)
	if err != nil {
		return resultado, err
	}
	defer linhas.Close()
	for linhas.Next() {
		var nota NotaFela
		if err = linhas.Scan(&nota.ID, &nota.AutorID, &nota.Nome, &nota.Nick, &nota.FotoVersao, &nota.Conteudo, &nota.CriadaEm); err != nil {
			return resultado, err
		}
		nota.PodeExcluir = visitante == perfil || visitante == nota.AutorID
		resultado.Notas = append(resultado.Notas, nota)
	}
	if err = linhas.Err(); err != nil {
		return resultado, err
	}
	if len(resultado.Notas) > 20 {
		resultado.Notas = resultado.Notas[:20]
		resultado.Proxima = resultado.Notas[19].ID
	}
	return resultado, nil
}

func CriarNotaFela(db *sql.DB, perfil, autor uint64, conteudo string) error {
	if perfil == autor {
		return ErrNotasIndisponiveis
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = travarUsuarios(tx, perfil, autor); err != nil {
		return err
	}
	var membro int
	err = tx.QueryRow("select 1 from felas where dono_id = ? and membro_id = ? for update", perfil, autor).Scan(&membro)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotasIndisponiveis
	}
	if err != nil {
		return err
	}
	var permitido bool
	err = tx.QueryRow(`
		select exists(select 1 from usuarios u
			join felas f on f.dono_id = u.id and f.membro_id = ?
			join usuarios a on a.id = f.membro_id
			where u.id = ? and u.notas_felas_ativas = 1 and u.suspenso_em is null and a.suspenso_em is null
			and not exists(select 1 from bloqueios b where (b.bloqueador_id = u.id and b.bloqueado_id = ?) or (b.bloqueador_id = ? and b.bloqueado_id = u.id))
			and (u.privado = 0 or exists(select 1 from seguidores s where s.usuario_id = u.id and s.seguidor_id = ?)))`,
		autor, perfil, autor, autor, autor).Scan(&permitido)
	if err != nil {
		return err
	}
	if !permitido {
		return ErrNotasIndisponiveis
	}
	if _, err = tx.Exec("insert into notas_felas (perfil_id, autor_id, conteudo) values (?, ?, ?)", perfil, autor, conteudo); err != nil {
		return err
	}
	return tx.Commit()
}

func ExcluirNotaFela(db *sql.DB, id, usuario uint64) error {
	resultado, err := db.Exec("delete from notas_felas where id = ? and (perfil_id = ? or autor_id = ?)", id, usuario, usuario)
	if err != nil {
		return err
	}
	afetadas, err := resultado.RowsAffected()
	if err != nil {
		return err
	}
	if afetadas == 0 {
		return ErrNotasIndisponiveis
	}
	return nil
}
