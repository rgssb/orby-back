package repositorios

import (
	"API/src/eventos"
	"API/src/modelos"
	"database/sql"
	"errors"
)

// Erros de regra de negócio das mensagens diretas
var (
	ErrMensagemParaSiMesmo = errors.New("Você não pode enviar mensagem para si mesmo")
	ErrUsuarioInexistente  = errors.New("Usuário não encontrado")
	ErrAguardandoAceite    = errors.New("Aguarde a pessoa responder antes de enviar mais mensagens")
	ErrBloqueado           = errors.New("Não é possível trocar mensagens com esta pessoa")
	ErrPedidoEncerrado     = errors.New("Esta pessoa encerrou o seu pedido")
	ErrRespostaInvalida    = errors.New("A mensagem respondida não pertence a esta conversa")
	ErrMensagemInexistente = errors.New("Mensagem não encontrada")
)

// LimiteAntesDoAceite é quantas mensagens seguidas se pode mandar sem resposta (pedido de conversa ou de desculpas)
const LimiteAntesDoAceite = 3

const (
	desculpaNenhuma   = 0
	desculpaAberta    = 1
	desculpaEncerrada = 2
)

// Mensagens representa um repositorio de mensagens diretas
type Mensagens struct {
	db *sql.DB
}

// NovoRepositorioDeMensagens cria um repositorio de mensagens
func NovoRepositorioDeMensagens(db *sql.DB) *Mensagens {
	return &Mensagens{db: db}
}

type consultador interface {
	QueryRow(query string, args ...any) *sql.Row
}

type estadoDaConversa struct {
	id           uint64
	existe       bool
	mutuo        bool
	aceita       bool
	iniciadaPor  uint64
	desculpa     int
	euBloqueei   bool
	fuiBloqueado bool
	limpa        uint64 // última mensagem que apaguei da minha visão
	ultimoID     uint64
}

func ordenar(a, b uint64) (uint64, uint64) {
	if a < b {
		return a, b
	}
	return b, a
}

// estado informa se a conversa existe, se foi aceita, se há bloqueio e se as duas pessoas se seguem
func estado(consulta consultador, eu, outro uint64) (estadoDaConversa, error) {
	var e estadoDaConversa
	erro := consulta.QueryRow(`
		select
			exists(select 1 from seguidores s where s.usuario_id = ? and s.seguidor_id = ?)
			and exists(select 1 from seguidores s where s.usuario_id = ? and s.seguidor_id = ?),
			exists(select 1 from bloqueios where bloqueador_id = ? and bloqueado_id = ?),
			exists(select 1 from bloqueios where bloqueador_id = ? and bloqueado_id = ?)`,
		eu, outro, outro, eu, eu, outro, outro, eu,
	).Scan(&e.mutuo, &e.euBloqueei, &e.fuiBloqueado)
	if erro != nil {
		return e, erro
	}

	menor, maior := ordenar(eu, outro)
	erro = consulta.QueryRow(`
		select id, aceita, iniciada_por, desculpa, if(usuario_menor = ?, limpa_menor, limpa_maior),
			coalesce((select max(m.id) from mensagens m where m.conversa_id = conversas.id), 0)
		from conversas where usuario_menor = ? and usuario_maior = ?`,
		eu, menor, maior,
	).Scan(&e.id, &e.aceita, &e.iniciadaPor, &e.desculpa, &e.limpa, &e.ultimoID)
	if errors.Is(erro, sql.ErrNoRows) {
		return e, nil
	}
	e.existe = erro == nil
	return e, erro
}

// marcar preenche no resumo da conversa o que a interface precisa saber sobre bloqueio e pedidos
func marcar(c *modelos.Conversa, eu uint64, e estadoDaConversa) {
	c.Bloqueei = e.euBloqueei
	c.Bloqueado = e.fuiBloqueado
	switch {
	case e.euBloqueei:
		c.Desculpa = e.desculpa == desculpaAberta && !e.fuiBloqueado
	case e.fuiBloqueado:
	default:
		pendente := !e.aceita && !e.mutuo
		c.Solicitacao = pendente && e.iniciadaPor != eu
		c.Aguardando = pendente && e.iniciadaPor == eu
	}
}

// Listar devolve as conversas do usuário, da mais recente para a mais antiga
func (repositorio Mensagens) Listar(eu uint64) ([]modelos.Conversa, error) {
	const minhaLimpeza = "if(c.usuario_menor = ?, c.limpa_menor, c.limpa_maior)"
	linhas, erro := repositorio.db.Query(`
		select o.id, o.nome, o.nick, `+colunaFotoVersao+`,
			m.texto, m.remetente_id = ?, m.criadoEm,
			(select count(*) from mensagens x where x.conversa_id = c.id and x.remetente_id <> ? and x.lida = false
				and x.id > `+minhaLimpeza+`),
			c.aceita, c.iniciada_por, c.desculpa, if(c.usuario_menor = ?, c.silenciada_menor, c.silenciada_maior),
			(exists(select 1 from seguidores s where s.usuario_id = c.usuario_menor and s.seguidor_id = c.usuario_maior)
			 and exists(select 1 from seguidores s where s.usuario_id = c.usuario_maior and s.seguidor_id = c.usuario_menor)),
			exists(select 1 from bloqueios b where b.bloqueador_id = ? and b.bloqueado_id = o.id),
			exists(select 1 from bloqueios b where b.bloqueador_id = o.id and b.bloqueado_id = ?)
		from conversas c
		join usuarios o on o.id = if(c.usuario_menor = ?, c.usuario_maior, c.usuario_menor)
		left join fotos_usuarios f on f.usuario_id = o.id
		join mensagens m on m.id = (select max(id) from mensagens where conversa_id = c.id)
		where (c.usuario_menor = ? or c.usuario_maior = ?) and m.id > `+minhaLimpeza+`
		order by m.id desc limit 100`,
		eu, eu, eu, eu, eu, eu, eu, eu, eu, eu,
	)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	conversas := []modelos.Conversa{}
	for linhas.Next() {
		var c modelos.Conversa
		var e estadoDaConversa
		if erro = linhas.Scan(&c.UsuarioID, &c.Nome, &c.Nick, &c.FotoVersao, &c.UltimaMensagem, &c.UltimaMinha,
			&c.AtualizadoEm, &c.NaoLidas, &e.aceita, &e.iniciadaPor, &e.desculpa, &c.Silenciada, &e.mutuo, &e.euBloqueei, &e.fuiBloqueado); erro != nil {
			return nil, erro
		}
		if c.UltimaMensagem == "" {
			c.UltimaMensagem = "📷 Imagem"
		}
		marcar(&c, eu, e)
		conversas = append(conversas, c)
	}
	return conversas, linhas.Err()
}

// ContarNaoLidas devolve as mensagens não lidas de conversas normais e o número de pedidos novos
func (repositorio Mensagens) ContarNaoLidas(eu uint64) (mensagens, solicitacoes uint64, erro error) {
	conversas, erro := repositorio.Listar(eu)
	if erro != nil {
		return 0, 0, erro
	}
	for _, c := range conversas {
		switch {
		case c.Silenciada:
		case c.Solicitacao || c.Desculpa:
			if c.NaoLidas > 0 {
				solicitacoes++
			}
		default:
			mensagens += c.NaoLidas
		}
	}
	return mensagens, solicitacoes, nil
}

// Abrir devolve o cabeçalho e as mensagens com outra pessoa (somente as novas, quando apos > 0) e marca as recebidas como lidas
func (repositorio Mensagens) Abrir(eu, outro, apos, antes uint64) (modelos.Dialogo, error) {
	dialogo := modelos.Dialogo{Mensagens: []modelos.Mensagem{}}

	erro := repositorio.db.QueryRow(
		"select u.id, u.nome, u.nick, "+colunaFotoVersao+" from usuarios u left join fotos_usuarios f on f.usuario_id = u.id where u.id = ?", outro,
	).Scan(&dialogo.UsuarioID, &dialogo.Nome, &dialogo.Nick, &dialogo.FotoVersao)
	if errors.Is(erro, sql.ErrNoRows) {
		return dialogo, ErrUsuarioInexistente
	}
	if erro != nil {
		return dialogo, erro
	}

	e, erro := estado(repositorio.db, eu, outro)
	if erro != nil {
		return dialogo, erro
	}
	if !e.existe || e.ultimoID <= e.limpa {
		dialogo.Bloqueei, dialogo.Bloqueado = e.euBloqueei, e.fuiBloqueado
		return dialogo, nil
	}
	marcar(&dialogo.Conversa, eu, e)
	if erro = repositorio.db.QueryRow(
		"select if(usuario_menor = ?, silenciada_menor, silenciada_maior) from conversas where id = ?", eu, e.id,
	).Scan(&dialogo.Silenciada); erro != nil {
		return dialogo, erro
	}
	if apos < e.limpa {
		apos = e.limpa
	}

	const limite = 50
	limiteSuperior := antes
	if limiteSuperior == 0 {
		limiteSuperior = 1 << 62
	}
	linhas, erro := repositorio.db.Query(`
		select * from (
			select m.id as mid, m.texto, m.remetente_id = ?, m.lida, m.criadoEm,
				exists(select 1 from imagens_mensagens i where i.mensagem_id = m.id),
				r.id as rid, r.texto as rtexto, r.remetente_id = ?, exists(select 1 from imagens_mensagens i where i.mensagem_id = r.id)
			from mensagens m left join mensagens r on r.id = m.responde_a
			where m.conversa_id = ? and m.id > ? and m.id < ? order by m.id desc limit ?
		) t order by mid asc`,
		eu, eu, e.id, apos, limiteSuperior, limite+1,
	)
	if erro != nil {
		return dialogo, erro
	}
	defer linhas.Close()

	indices := map[uint64]int{}
	for linhas.Next() {
		var m modelos.Mensagem
		var citadaID sql.NullInt64
		var citadaTexto sql.NullString
		var citadaMinha, citadaImagem sql.NullBool
		if erro = linhas.Scan(&m.ID, &m.Texto, &m.Minha, &m.Lida, &m.CriadoEm, &m.TemImagem,
			&citadaID, &citadaTexto, &citadaMinha, &citadaImagem); erro != nil {
			return dialogo, erro
		}
		if citadaID.Valid {
			m.RespondeA = &modelos.Citacao{ID: uint64(citadaID.Int64), Texto: citadaTexto.String,
				Minha: citadaMinha.Bool, TemImagem: citadaImagem.Bool}
		}
		m.Reacoes = []modelos.Reacao{}
		dialogo.Mensagens = append(dialogo.Mensagens, m)
	}
	if erro = linhas.Err(); erro != nil {
		return dialogo, erro
	}
	linhas.Close()

	if len(dialogo.Mensagens) > limite {
		dialogo.TemMais = true
		dialogo.Mensagens = dialogo.Mensagens[1:]
	}
	for i, m := range dialogo.Mensagens {
		indices[m.ID] = i
	}

	if len(dialogo.Mensagens) > 0 {
		reacoes, erro := repositorio.db.Query(`
			select mensagem_id, emoji, count(*), max(usuario_id = ?) from reacoes_mensagens
			where mensagem_id between ? and ? group by mensagem_id, emoji order by min(criadoEm)`,
			eu, dialogo.Mensagens[0].ID, dialogo.Mensagens[len(dialogo.Mensagens)-1].ID,
		)
		if erro != nil {
			return dialogo, erro
		}
		defer reacoes.Close()
		for reacoes.Next() {
			var id uint64
			var r modelos.Reacao
			if erro = reacoes.Scan(&id, &r.Emoji, &r.Total, &r.Minha); erro != nil {
				return dialogo, erro
			}
			if i, ok := indices[id]; ok {
				dialogo.Mensagens[i].Reacoes = append(dialogo.Mensagens[i].Reacoes, r)
			}
		}
	}

	resultado, erro := repositorio.db.Exec(
		"update mensagens set lida = true where conversa_id = ? and remetente_id <> ? and lida = false", e.id, eu,
	)
	if erro != nil {
		return dialogo, erro
	}
	if lidas, _ := resultado.RowsAffected(); lidas > 0 {
		eventos.Publicar(outro, eventos.Evento{Tipo: "lido", De: eu})
	}
	return dialogo, nil
}

// Enviar grava uma mensagem. Sem seguimento mútuo ela vira um pedido de conversa; com bloqueio, só
// a pessoa bloqueada pode mandar um pedido de desculpas e só quem bloqueou pode responder a ele.
func (repositorio Mensagens) Enviar(eu, outro uint64, texto string, respondeA uint64, comImagem bool) (modelos.Mensagem, error) {
	var mensagem modelos.Mensagem
	if eu == outro {
		return mensagem, ErrMensagemParaSiMesmo
	}

	var existe bool
	if erro := repositorio.db.QueryRow("select exists(select 1 from usuarios where id = ?)", outro).Scan(&existe); erro != nil {
		return mensagem, erro
	}
	if !existe {
		return mensagem, ErrUsuarioInexistente
	}

	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return mensagem, erro
	}
	defer transacao.Rollback()

	menor, maior := ordenar(eu, outro)
	if _, erro = transacao.Exec(
		"insert ignore into conversas (usuario_menor, usuario_maior, iniciada_por) values (?, ?, ?)", menor, maior, eu,
	); erro != nil {
		return mensagem, erro
	}

	var conversaID uint64
	if erro = transacao.QueryRow(
		"select id from conversas where usuario_menor = ? and usuario_maior = ? for update", menor, maior,
	).Scan(&conversaID); erro != nil {
		return mensagem, erro
	}

	e, erro := estado(transacao, eu, outro)
	if erro != nil {
		return mensagem, erro
	}

	switch {
	case e.euBloqueei && e.fuiBloqueado:
		return mensagem, ErrBloqueado
	case e.euBloqueei:
		// Quem bloqueou só responde a um pedido de desculpas aberto
		if e.desculpa != desculpaAberta {
			return mensagem, ErrBloqueado
		}
	case e.fuiBloqueado:
		if e.desculpa == desculpaEncerrada {
			return mensagem, ErrPedidoEncerrado
		}
		var semResposta int
		if erro = transacao.QueryRow(`
			select count(*) from mensagens
			where conversa_id = ? and remetente_id = ?
			and id > coalesce((select max(id) from mensagens where conversa_id = ? and remetente_id <> ?), 0)`,
			conversaID, eu, conversaID, eu,
		).Scan(&semResposta); erro != nil {
			return mensagem, erro
		}
		if semResposta >= LimiteAntesDoAceite {
			return mensagem, ErrAguardandoAceite
		}
		if e.desculpa == desculpaNenhuma {
			if _, erro = transacao.Exec("update conversas set desculpa = ? where id = ?", desculpaAberta, conversaID); erro != nil {
				return mensagem, erro
			}
		}
	default:
		if !e.aceita {
			if e.mutuo || e.iniciadaPor != eu {
				// Seguimento mútuo ou resposta de quem recebeu o pedido: a conversa passa a ser normal
				if _, erro = transacao.Exec("update conversas set aceita = true where id = ?", conversaID); erro != nil {
					return mensagem, erro
				}
			} else {
				var enviadas int
				if erro = transacao.QueryRow(`
					select count(*) from mensagens
					where conversa_id = ? and remetente_id = ?
					and id > coalesce((select max(id) from mensagens where conversa_id = ? and remetente_id <> ?), 0)`,
					conversaID, eu, conversaID, eu,
				).Scan(&enviadas); erro != nil {
					return mensagem, erro
				}
				if enviadas >= LimiteAntesDoAceite {
					return mensagem, ErrAguardandoAceite
				}
			}
		}
	}

	var citada sql.NullInt64
	if respondeA > 0 {
		var pertence bool
		if erro = transacao.QueryRow("select exists(select 1 from mensagens where id = ? and conversa_id = ?)", respondeA, conversaID).Scan(&pertence); erro != nil {
			return mensagem, erro
		}
		if !pertence {
			return mensagem, ErrRespostaInvalida
		}
		citada = sql.NullInt64{Int64: int64(respondeA), Valid: true}
	}

	resultado, erro := transacao.Exec(
		"insert into mensagens (conversa_id, remetente_id, texto, responde_a) values (?, ?, ?, ?)", conversaID, eu, texto, citada,
	)
	if erro != nil {
		return mensagem, erro
	}
	id, erro := resultado.LastInsertId()
	if erro != nil {
		return mensagem, erro
	}

	if erro = transacao.QueryRow("select criadoEm from mensagens where id = ?", id).Scan(&mensagem.CriadoEm); erro != nil {
		return mensagem, erro
	}
	mensagem.ID, mensagem.Texto, mensagem.Minha, mensagem.TemImagem = uint64(id), texto, true, comImagem
	mensagem.Reacoes = []modelos.Reacao{}

	if erro = transacao.Commit(); erro != nil {
		return mensagem, erro
	}
	// Com imagem, o aviso sai quando ela terminar de subir
	if !comImagem {
		eventos.Publicar(outro, eventos.Evento{Tipo: "mensagem", De: eu})
	}
	return mensagem, nil
}

// Aceitar libera uma conversa recebida como pedido; devolve false se não havia pedido a aceitar
func (repositorio Mensagens) Aceitar(eu, outro uint64) (bool, error) {
	e, erro := estado(repositorio.db, eu, outro)
	if erro != nil || !e.existe || e.euBloqueei || e.fuiBloqueado || e.iniciadaPor == eu {
		return false, erro
	}

	_, erro = repositorio.db.Exec("update conversas set aceita = true where id = ?", e.id)
	return erro == nil, erro
}

// ExcluirConversa some com a conversa só para quem a apagou; a outra pessoa continua vendo tudo.
// Se quem bloqueou apaga um pedido de desculpas, o pedido é encerrado e a pessoa não pode insistir.
func (repositorio Mensagens) ExcluirConversa(eu, outro uint64) error {
	menor, maior := ordenar(eu, outro)
	_, erro := repositorio.db.Exec(`
		update conversas set
			limpa_menor = if(usuario_menor = ?, coalesce((select max(m.id) from mensagens m where m.conversa_id = conversas.id), 0), limpa_menor),
			limpa_maior = if(usuario_maior = ?, coalesce((select max(m.id) from mensagens m where m.conversa_id = conversas.id), 0), limpa_maior),
			desculpa = if(desculpa = ? and exists(select 1 from bloqueios where bloqueador_id = ? and bloqueado_id = ?), ?, desculpa)
		where usuario_menor = ? and usuario_maior = ?`,
		eu, eu, desculpaAberta, eu, outro, desculpaEncerrada, menor, maior,
	)
	if erro != nil {
		return erro
	}

	// Quando as duas pessoas já apagaram tudo e não há bloqueio, não sobra nada a guardar
	_, erro = repositorio.db.Exec(`
		delete from conversas
		where usuario_menor = ? and usuario_maior = ?
		and limpa_menor >= coalesce((select max(m.id) from mensagens m where m.conversa_id = conversas.id), 0)
		and limpa_maior >= coalesce((select max(m.id) from mensagens m where m.conversa_id = conversas.id), 0)
		and not exists(select 1 from bloqueios where (bloqueador_id = ? and bloqueado_id = ?) or (bloqueador_id = ? and bloqueado_id = ?))`,
		menor, maior, eu, outro, outro, eu,
	)
	return erro
}

// ExcluirMensagem apaga uma mensagem enviada pelo próprio usuário; devolve false se não for dele
func (repositorio Mensagens) ExcluirMensagem(eu, mensagemID uint64) (bool, error) {
	destino, erro := repositorio.outroDaMensagem(eu, mensagemID)
	if erro != nil || destino == 0 {
		return false, erro
	}
	resultado, erro := repositorio.db.Exec("delete from mensagens where id = ? and remetente_id = ?", mensagemID, eu)
	if erro != nil {
		return false, erro
	}
	linhas, erro := resultado.RowsAffected()
	if linhas > 0 {
		eventos.Publicar(destino, eventos.Evento{Tipo: "mensagem", De: eu})
	}
	return linhas > 0, erro
}

// outroDaMensagem devolve a outra pessoa da conversa quando eu participo dela (e, em ExcluirMensagem, sou o autor); 0 se não
func (repositorio Mensagens) outroDaMensagem(eu, mensagemID uint64) (uint64, error) {
	var menor, maior, remetente uint64
	erro := repositorio.db.QueryRow(`
		select c.usuario_menor, c.usuario_maior, m.remetente_id from mensagens m
		join conversas c on c.id = m.conversa_id where m.id = ?`, mensagemID,
	).Scan(&menor, &maior, &remetente)
	if errors.Is(erro, sql.ErrNoRows) {
		return 0, nil
	}
	if erro != nil || (eu != menor && eu != maior) {
		return 0, erro
	}
	if menor == eu {
		return maior, nil
	}
	return menor, nil
}

// participante devolve a outra pessoa da mensagem, ou ErrMensagemInexistente se eu não participo da conversa
func (repositorio Mensagens) participante(eu, mensagemID uint64) (uint64, error) {
	outro, erro := repositorio.outroDaMensagem(eu, mensagemID)
	if erro == nil && outro == 0 {
		erro = ErrMensagemInexistente
	}
	return outro, erro
}

// Reagir alterna a reação do usuário a uma mensagem: repetir o mesmo emoji remove, outro emoji troca
func (repositorio Mensagens) Reagir(eu, mensagemID uint64, emoji string) error {
	outro, erro := repositorio.participante(eu, mensagemID)
	if erro != nil {
		return erro
	}
	var atual string
	erro = repositorio.db.QueryRow("select emoji from reacoes_mensagens where mensagem_id = ? and usuario_id = ?", mensagemID, eu).Scan(&atual)
	switch {
	case errors.Is(erro, sql.ErrNoRows):
		_, erro = repositorio.db.Exec("insert into reacoes_mensagens (mensagem_id, usuario_id, emoji) values (?, ?, ?)", mensagemID, eu, emoji)
	case erro != nil:
	case atual == emoji:
		_, erro = repositorio.db.Exec("delete from reacoes_mensagens where mensagem_id = ? and usuario_id = ?", mensagemID, eu)
	default:
		_, erro = repositorio.db.Exec("update reacoes_mensagens set emoji = ? where mensagem_id = ? and usuario_id = ?", emoji, mensagemID, eu)
	}
	if erro == nil {
		eventos.Publicar(outro, eventos.Evento{Tipo: "mensagem", De: eu})
	}
	return erro
}

// SalvarImagem anexa a imagem a uma mensagem enviada pelo próprio usuário
func (repositorio Mensagens) SalvarImagem(eu, mensagemID uint64, tipo string, dados []byte) error {
	var remetente uint64
	if erro := repositorio.db.QueryRow("select remetente_id from mensagens where id = ?", mensagemID).Scan(&remetente); erro != nil || remetente != eu {
		if erro == nil || errors.Is(erro, sql.ErrNoRows) {
			erro = ErrMensagemInexistente
		}
		return erro
	}
	outro, erro := repositorio.participante(eu, mensagemID)
	if erro != nil {
		return erro
	}
	if _, erro = repositorio.db.Exec(
		"insert into imagens_mensagens (mensagem_id, tipo, dados) values (?, ?, ?) on duplicate key update tipo = values(tipo), dados = values(dados)",
		mensagemID, tipo, dados,
	); erro != nil {
		return erro
	}
	eventos.Publicar(outro, eventos.Evento{Tipo: "mensagem", De: eu})
	return nil
}

// BuscarImagem devolve a imagem de uma mensagem para quem participa da conversa
func (repositorio Mensagens) BuscarImagem(eu, mensagemID uint64) (string, []byte, error) {
	if _, erro := repositorio.participante(eu, mensagemID); erro != nil {
		return "", nil, erro
	}
	var tipo string
	var dados []byte
	erro := repositorio.db.QueryRow("select tipo, dados from imagens_mensagens where mensagem_id = ?", mensagemID).Scan(&tipo, &dados)
	if errors.Is(erro, sql.ErrNoRows) {
		erro = ErrMensagemInexistente
	}
	return tipo, dados, erro
}

// Silenciar liga ou desliga o silêncio da conversa só para o usuário
func (repositorio Mensagens) Silenciar(eu, outro uint64, silenciar bool) error {
	menor, maior := ordenar(eu, outro)
	_, erro := repositorio.db.Exec(`
		update conversas set
			silenciada_menor = if(usuario_menor = ?, ?, silenciada_menor),
			silenciada_maior = if(usuario_maior = ?, ?, silenciada_maior)
		where usuario_menor = ? and usuario_maior = ?`,
		eu, silenciar, eu, silenciar, menor, maior,
	)
	return erro
}

// Digitando avisa a outra pessoa, em tempo real, que o usuário está escrevendo
func (repositorio Mensagens) Digitando(eu, outro uint64) {
	eventos.Publicar(outro, eventos.Evento{Tipo: "digitando", De: eu})
}
