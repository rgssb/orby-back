package repositorios

import (
	"database/sql"
	"time"
)

type DetalheDenuncia struct {
	Tipo         string    `json:"tipo"`
	AlvoID       uint64    `json:"alvoId"`
	PublicacaoID uint64    `json:"publicacaoId"`
	Motivo       string    `json:"motivo"`
	Resolvida    bool      `json:"resolvida"`
	Total        uint64    `json:"total"`
	UltimaEm     time.Time `json:"ultimaEm"`
}

type AcaoModeracao struct {
	Acao  string    `json:"acao"`
	Admin string    `json:"admin"`
	Data  time.Time `json:"data"`
}

func registrarAcaoModeracao(tx *sql.Tx, adminID, alvoID uint64, acao string) error {
	_, erro := tx.Exec("insert into acoes_moderacao (admin_id, alvo_id, acao) values (?, ?, ?)", adminID, alvoID, acao)
	return erro
}

func (repositorio Denuncias) Detalhes(donoID, deslocamento uint64) ([]DetalheDenuncia, error) {
	linhas, erro := repositorio.db.Query(`
		select d.tipo, d.alvo_id,
			case when d.tipo = 'publicacao' then coalesce(p.id, 0)
				when d.tipo = 'comentario' then coalesce(c.publicacao_id, 0) else 0 end,
			d.motivo, d.resolvida, count(*), max(d.criadoEm)
		from denuncias d
		left join publicacoes p on d.tipo = 'publicacao' and p.id = d.alvo_id
		left join comentarios c on d.tipo = 'comentario' and c.id = d.alvo_id
		where d.dono_id = ?
		group by d.tipo, d.alvo_id, d.motivo, d.resolvida, p.id, c.publicacao_id
		order by d.resolvida asc, max(d.criadoEm) desc, d.alvo_id desc, d.tipo, d.motivo
		limit 16 offset ?`, donoID, deslocamento)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	detalhes := []DetalheDenuncia{}
	for linhas.Next() {
		var d DetalheDenuncia
		if erro = linhas.Scan(&d.Tipo, &d.AlvoID, &d.PublicacaoID, &d.Motivo, &d.Resolvida, &d.Total, &d.UltimaEm); erro != nil {
			return nil, erro
		}
		detalhes = append(detalhes, d)
	}
	return detalhes, linhas.Err()
}

func (repositorio Denuncias) AcoesRecentes(donoID uint64) ([]AcaoModeracao, error) {
	linhas, erro := repositorio.db.Query(`
		select a.acao, coalesce(u.nick, 'Conta removida'), a.criadoEm
		from acoes_moderacao a left join usuarios u on u.id = a.admin_id
		where a.alvo_id = ? order by a.id desc limit 30`, donoID)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	acoes := []AcaoModeracao{}
	for linhas.Next() {
		var a AcaoModeracao
		if erro = linhas.Scan(&a.Acao, &a.Admin, &a.Data); erro != nil {
			return nil, erro
		}
		acoes = append(acoes, a)
	}
	return acoes, linhas.Err()
}
