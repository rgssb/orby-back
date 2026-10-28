package repositorios

import (
	"errors"
	"orby/api/src/modelos"
)

var ErrPreferenciasInvalidas = errors.New("Preferências de notificação inválidas")

var tiposConfiguraveis = map[string]bool{
	modelos.NotificacaoCurtidaPublicacao: true,
	modelos.NotificacaoCurtidaComentario: true,
	modelos.NotificacaoComentario:        true,
	modelos.NotificacaoResposta:          true,
	modelos.NotificacaoRepost:            true,
	modelos.NotificacaoMencao:            true,
	modelos.NotificacaoSeguidor:          true,
}

func (repositorio Notificacoes) TiposSilenciados(usuarioID uint64) ([]string, error) {
	linhas, erro := repositorio.db.Query("select tipo from notificacoes_silenciadas where usuario_id = ? order by tipo", usuarioID)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	tipos := []string{}
	for linhas.Next() {
		var tipo string
		if erro = linhas.Scan(&tipo); erro != nil {
			return nil, erro
		}
		tipos = append(tipos, tipo)
	}
	return tipos, linhas.Err()
}

func (repositorio Notificacoes) SalvarTiposSilenciados(usuarioID uint64, tipos []string) error {
	if len(tipos) > len(tiposConfiguraveis) {
		return ErrPreferenciasInvalidas
	}
	vistos := make(map[string]bool, len(tipos))
	for _, tipo := range tipos {
		if !tiposConfiguraveis[tipo] || vistos[tipo] {
			return ErrPreferenciasInvalidas
		}
		vistos[tipo] = true
	}
	tx, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer tx.Rollback()
	if _, erro = tx.Exec("delete from notificacoes_silenciadas where usuario_id = ?", usuarioID); erro != nil {
		return erro
	}
	for _, tipo := range tipos {
		if _, erro = tx.Exec("insert into notificacoes_silenciadas (usuario_id, tipo) values (?, ?)", usuarioID, tipo); erro != nil {
			return erro
		}
	}
	return tx.Commit()
}
