package repositorios

import (
	"database/sql"
	"errors"
)

var ErrPastaNaoEncontrada = errors.New("Pasta ou publicação salva não encontrada")

type PastaSalvos struct {
	ID       uint64 `json:"id"`
	Nome     string `json:"nome"`
	Incluida bool   `json:"incluida"`
}

func (repositorio Publicacoes) CriarPasta(usuarioID uint64, nome string) (uint64, error) {
	resultado, erro := repositorio.db.Exec("insert into pastas_salvos (usuario_id, nome) values (?, ?)", usuarioID, nome)
	if erro != nil {
		return 0, erro
	}
	id, erro := resultado.LastInsertId()
	return uint64(id), erro
}

func (repositorio Publicacoes) CriarPastaParaPublicacao(usuarioID, publicacaoID uint64, nome string) (uint64, error) {
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return 0, erro
	}
	defer tx.Rollback()
	var id uint64
	erro = tx.QueryRow(`
		select sv.publicacao_id from salvos sv
		join publicacoes p on p.id = sv.publicacao_id
		join usuarios u on u.id = p.autor_id
		where sv.usuario_id = ? and sv.publicacao_id = ? and `+postVisivel+` for update`,
		usuarioID, publicacaoID, usuarioID, usuarioID, usuarioID, usuarioID).Scan(&id)
	if errors.Is(erro, sql.ErrNoRows) {
		return 0, ErrPastaNaoEncontrada
	}
	if erro != nil {
		return 0, erro
	}
	resultado, erro := tx.Exec("insert into pastas_salvos (usuario_id, nome) values (?, ?)", usuarioID, nome)
	if erro != nil {
		return 0, erro
	}
	criada, erro := resultado.LastInsertId()
	if erro != nil {
		return 0, erro
	}
	if _, erro = tx.Exec("insert into pastas_publicacoes (pasta_id, usuario_id, publicacao_id) values (?, ?, ?)", criada, usuarioID, publicacaoID); erro != nil {
		return 0, erro
	}
	if erro = tx.Commit(); erro != nil {
		return 0, erro
	}
	return uint64(criada), nil
}

func (repositorio Publicacoes) ListarPastas(usuarioID, publicacaoID uint64) ([]PastaSalvos, error) {
	linhas, erro := repositorio.db.Query(`
		select p.id, p.nome, exists(
			select 1 from pastas_publicacoes pp
			where pp.pasta_id = p.id and pp.usuario_id = p.usuario_id and pp.publicacao_id = ?
		)
		from pastas_salvos p where p.usuario_id = ? order by p.nome, p.id`, publicacaoID, usuarioID)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	pastas := []PastaSalvos{}
	for linhas.Next() {
		var pasta PastaSalvos
		if erro = linhas.Scan(&pasta.ID, &pasta.Nome, &pasta.Incluida); erro != nil {
			return nil, erro
		}
		pastas = append(pastas, pasta)
	}
	return pastas, linhas.Err()
}

func (repositorio Publicacoes) PastaExiste(usuarioID, pastaID uint64) (bool, error) {
	var existe bool
	erro := repositorio.db.QueryRow("select exists(select 1 from pastas_salvos where id = ? and usuario_id = ?)", pastaID, usuarioID).Scan(&existe)
	return existe, erro
}

func (repositorio Publicacoes) RenomearPasta(usuarioID, pastaID uint64, nome string) error {
	resultado, erro := repositorio.db.Exec("update pastas_salvos set nome = ? where id = ? and usuario_id = ?", nome, pastaID, usuarioID)
	if erro != nil {
		return erro
	}
	afetadas, erro := resultado.RowsAffected()
	if erro != nil {
		return erro
	}
	if afetadas == 0 {
		existe, erro := repositorio.PastaExiste(usuarioID, pastaID)
		if erro != nil {
			return erro
		}
		if !existe {
			return ErrPastaNaoEncontrada
		}
	}
	return nil
}

func (repositorio Publicacoes) ExcluirPasta(usuarioID, pastaID uint64) error {
	resultado, erro := repositorio.db.Exec("delete from pastas_salvos where id = ? and usuario_id = ?", pastaID, usuarioID)
	if erro != nil {
		return erro
	}
	afetadas, erro := resultado.RowsAffected()
	if erro != nil {
		return erro
	}
	if afetadas == 0 {
		return ErrPastaNaoEncontrada
	}
	return nil
}

func (repositorio Publicacoes) DefinirPastas(usuarioID, publicacaoID uint64, pastas []uint64) error {
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	var id uint64
	erro = tx.QueryRow(`
		select sv.publicacao_id from salvos sv
		join publicacoes p on p.id = sv.publicacao_id
		join usuarios u on u.id = p.autor_id
		where sv.usuario_id = ? and sv.publicacao_id = ? and `+postVisivel+` for update`,
		usuarioID, publicacaoID, usuarioID, usuarioID, usuarioID, usuarioID).Scan(&id)
	if errors.Is(erro, sql.ErrNoRows) {
		return ErrPastaNaoEncontrada
	}
	if erro != nil {
		return erro
	}
	for _, pastaID := range pastas {
		erro = tx.QueryRow("select id from pastas_salvos where id = ? and usuario_id = ? for update", pastaID, usuarioID).Scan(&id)
		if errors.Is(erro, sql.ErrNoRows) {
			return ErrPastaNaoEncontrada
		}
		if erro != nil {
			return erro
		}
	}
	if _, erro = tx.Exec("delete from pastas_publicacoes where usuario_id = ? and publicacao_id = ?", usuarioID, publicacaoID); erro != nil {
		return erro
	}
	for _, pastaID := range pastas {
		if _, erro = tx.Exec("insert into pastas_publicacoes (pasta_id, usuario_id, publicacao_id) values (?, ?, ?)", pastaID, usuarioID, publicacaoID); erro != nil {
			return erro
		}
	}
	return tx.Commit()
}
