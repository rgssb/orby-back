package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/modelos"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var ErrVotoExistente = errors.New("Você já votou nesta enquete")
var ErrEnqueteIndisponivel = errors.New("Enquete encerrada, opção inválida ou publicação indisponível")

// VotarEnquete verifica acesso e prazo na própria operação que grava o voto.
func (repositorio Publicacoes) VotarEnquete(usuarioID, publicacaoID, opcaoID uint64) error {
	resultado, erro := repositorio.db.Exec(`
		insert into votos_enquetes (publicacao_id, usuario_id, opcao_id)
		select e.publicacao_id, ?, o.id
		from enquetes e
		join opcoes_enquetes o on o.publicacao_id = e.publicacao_id
		join publicacoes p on p.id = e.publicacao_id
		join usuarios u on u.id = p.autor_id
		where e.publicacao_id = ? and o.id = ?
			and (e.encerra_em is null or e.encerra_em > unix_timestamp())
			and `+postVisivel,
		usuarioID, publicacaoID, opcaoID, usuarioID, usuarioID, usuarioID, usuarioID)
	if erro != nil {
		var mysqlErro *mysql.MySQLError
		if errors.As(erro, &mysqlErro) && mysqlErro.Number == 1062 {
			return ErrVotoExistente
		}
		return erro
	}
	total, erro := resultado.RowsAffected()
	if erro != nil {
		return erro
	}
	if total == 0 {
		return ErrEnqueteIndisponivel
	}
	return nil
}

func (repositorio Publicacoes) anexarEnquetes(publicacoes []modelos.Publicacao, visitanteID uint64) error {
	ids := make([]interface{}, len(publicacoes))
	posicoes := make(map[uint64]int, len(publicacoes))
	for i, publicacao := range publicacoes {
		ids[i] = publicacao.ID
		posicoes[publicacao.ID] = i
	}
	if len(ids) == 0 {
		return nil
	}
	marcadores := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := append([]interface{}{visitanteID, visitanteID}, ids...)
	linhas, erro := repositorio.db.Query(`
		select e.publicacao_id, e.titulo, e.encerra_em,
			(e.encerra_em is not null and e.encerra_em <= unix_timestamp()),
			exists(select 1 from votos_enquetes meu where meu.publicacao_id = e.publicacao_id and meu.usuario_id = ?),
			o.id, o.texto, count(v.usuario_id), coalesce(max(v.usuario_id = ?), 0)
		from enquetes e
		join opcoes_enquetes o on o.publicacao_id = e.publicacao_id
		left join votos_enquetes v on v.publicacao_id = e.publicacao_id and v.opcao_id = o.id
		where e.publicacao_id in (`+marcadores+`)
		group by e.publicacao_id, e.titulo, e.encerra_em, o.id, o.texto
		order by e.publicacao_id, o.id`, args...)
	if erro != nil {
		return erro
	}
	defer linhas.Close()
	for linhas.Next() {
		var id, votos uint64
		var titulo string
		var prazo sql.NullInt64
		var encerrada, votou bool
		var opcao modelos.OpcaoEnquete
		if erro = linhas.Scan(&id, &titulo, &prazo, &encerrada, &votou, &opcao.ID, &opcao.Texto, &votos, &opcao.Minha); erro != nil {
			return erro
		}
		publicacao := &publicacoes[posicoes[id]]
		if publicacao.Enquete == nil {
			publicacao.Enquete = &modelos.Enquete{Titulo: titulo, Encerrada: encerrada, Votou: votou}
			if prazo.Valid {
				instante := time.Unix(prazo.Int64, 0).UTC()
				publicacao.Enquete.EncerraEm = &instante
			}
		}
		if votou || encerrada {
			opcao.Votos = votos
			publicacao.Enquete.TotalVotos += votos
		}
		publicacao.Enquete.Opcoes = append(publicacao.Enquete.Opcoes, opcao)
	}
	return linhas.Err()
}
