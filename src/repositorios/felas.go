package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
)

var ErrFelaInvalido = errors.New("Não é possível adicionar esta conta aos Felas")
var ErrFelasSemSeguimentoMutuo = errors.New("Para adicionar aos Felas, vocês precisam se seguir mutuamente")

type Felas struct{ db *sql.DB }

func NovoRepositorioDeFelas(db *sql.DB) *Felas { return &Felas{db: db} }

func (repositorio Felas) Listar(dono uint64) ([]modelos.SugestaoDeSeguir, error) {
	linhas, erro := repositorio.db.Query(`
		select u.id, u.nome, u.nick, u.privado, `+colunaFotoVersao+`
		from felas fl join usuarios u on u.id = fl.membro_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where fl.dono_id = ?
		order by u.nome, u.id`, dono)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	pessoas := []modelos.SugestaoDeSeguir{}
	for linhas.Next() {
		var pessoa modelos.SugestaoDeSeguir
		if erro = linhas.Scan(&pessoa.ID, &pessoa.Nome, &pessoa.Nick, &pessoa.Privado, &pessoa.FotoVersao); erro != nil {
			return nil, erro
		}
		pessoas = append(pessoas, pessoa)
	}
	return pessoas, linhas.Err()
}

func (repositorio Felas) Estado(dono, membro uint64) (bool, bool, error) {
	var existe, podeAdicionar bool
	erro := repositorio.db.QueryRow(`
		select exists(select 1 from felas where dono_id = ? and membro_id = ?),
			exists(select 1 from usuarios u where u.id = ? and u.id <> ? and u.suspenso_em is null
				and not exists (select 1 from bloqueios b where
					(b.bloqueador_id = ? and b.bloqueado_id = u.id) or
					(b.bloqueador_id = u.id and b.bloqueado_id = ?))
				and exists (select 1 from seguidores where usuario_id = ? and seguidor_id = u.id)
				and exists (select 1 from seguidores where usuario_id = u.id and seguidor_id = ?))`,
		dono, membro, membro, dono, dono, dono, dono, dono).Scan(&existe, &podeAdicionar)
	return existe, podeAdicionar, erro
}

func (repositorio Felas) Adicionar(dono, membro uint64) error {
	if dono == membro || membro == 0 {
		return ErrFelaInvalido
	}
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	if erro = travarUsuarios(tx, dono, membro); erro != nil {
		return erro
	}
	var permitido bool
	erro = tx.QueryRow(`
		select exists(select 1 from usuarios u where u.id = ? and u.suspenso_em is null
			and not exists (select 1 from bloqueios b where
				(b.bloqueador_id = ? and b.bloqueado_id = u.id) or
				(b.bloqueador_id = u.id and b.bloqueado_id = ?)))`, membro, dono, dono).Scan(&permitido)
	if erro != nil {
		return erro
	}
	if !permitido {
		return ErrFelaInvalido
	}
	linhas, erro := tx.Query(`
		select usuario_id from seguidores
		where (usuario_id = ? and seguidor_id = ?) or (usuario_id = ? and seguidor_id = ?)
		for update`, dono, membro, membro, dono)
	if erro != nil {
		return erro
	}
	var quantidade int
	for linhas.Next() {
		quantidade++
	}
	erro = linhas.Err()
	linhas.Close()
	if erro != nil {
		return erro
	}
	if quantidade != 2 {
		return ErrFelasSemSeguimentoMutuo
	}
	if _, erro = tx.Exec("insert ignore into felas (dono_id, membro_id) values (?, ?)", dono, membro); erro != nil {
		return erro
	}
	return tx.Commit()
}

func (repositorio Felas) Remover(dono, membro uint64) error {
	_, erro := repositorio.db.Exec("delete from felas where dono_id = ? and membro_id = ?", dono, membro)
	return erro
}

func travarUsuarios(tx *sql.Tx, primeiro, segundo uint64) error {
	linhas, erro := tx.Query("select id from usuarios where id in (?, ?) order by id for update", primeiro, segundo)
	if erro != nil {
		return erro
	}
	defer linhas.Close()
	for linhas.Next() {
		var id uint64
		if erro = linhas.Scan(&id); erro != nil {
			return erro
		}
	}
	return linhas.Err()
}
