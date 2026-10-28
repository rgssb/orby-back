package repositorios

import "database/sql"

// Bloqueios representa um repositorio de bloqueios entre usuarios
type Bloqueios struct {
	db *sql.DB
}

// NovoRepositorioDeBloqueios cria um repositorio de bloqueios
func NovoRepositorioDeBloqueios(db *sql.DB) *Bloqueios {
	return &Bloqueios{db: db}
}

// Bloquear impede os dois de se seguirem ou conversarem e desfaz o seguimento que existia
func (repositorio Bloqueios) Bloquear(bloqueadorID, bloqueadoID uint64) error {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer transacao.Rollback()

	if erro = travarUsuarios(transacao, bloqueadorID, bloqueadoID); erro != nil {
		return erro
	}
	if _, erro = transacao.Exec("insert ignore into bloqueios (bloqueador_id, bloqueado_id) values (?, ?)", bloqueadorID, bloqueadoID); erro != nil {
		return erro
	}
	if _, erro = transacao.Exec(
		"delete from seguidores where (usuario_id = ? and seguidor_id = ?) or (usuario_id = ? and seguidor_id = ?)",
		bloqueadorID, bloqueadoID, bloqueadoID, bloqueadorID,
	); erro != nil {
		return erro
	}
	if _, erro = transacao.Exec(
		"delete from felas where (dono_id = ? and membro_id = ?) or (dono_id = ? and membro_id = ?)",
		bloqueadorID, bloqueadoID, bloqueadoID, bloqueadorID,
	); erro != nil {
		return erro
	}

	menor, maior := ordenar(bloqueadorID, bloqueadoID)
	if _, erro = transacao.Exec("update conversas set desculpa = ? where usuario_menor = ? and usuario_maior = ?", desculpaNenhuma, menor, maior); erro != nil {
		return erro
	}
	return transacao.Commit()
}

// Desbloquear remove o bloqueio e zera o pedido de desculpas
func (repositorio Bloqueios) Desbloquear(bloqueadorID, bloqueadoID uint64) error {
	if _, erro := repositorio.db.Exec("delete from bloqueios where bloqueador_id = ? and bloqueado_id = ?", bloqueadorID, bloqueadoID); erro != nil {
		return erro
	}
	menor, maior := ordenar(bloqueadorID, bloqueadoID)
	_, erro := repositorio.db.Exec("update conversas set desculpa = ? where usuario_menor = ? and usuario_maior = ?", desculpaNenhuma, menor, maior)
	return erro
}

// Situacao diz se eu bloqueei a outra pessoa e se ela me bloqueou
func (repositorio Bloqueios) Situacao(eu, outro uint64) (bloqueei, bloqueado bool, erro error) {
	erro = repositorio.db.QueryRow(`
		select exists(select 1 from bloqueios where bloqueador_id = ? and bloqueado_id = ?),
			exists(select 1 from bloqueios where bloqueador_id = ? and bloqueado_id = ?)`,
		eu, outro, outro, eu,
	).Scan(&bloqueei, &bloqueado)
	return
}
