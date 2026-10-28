package repositorios

import (
	"database/sql"
	"errors"
	"orby/api/src/eventos"
	"orby/api/src/midia"
	"orby/api/src/modelos"
)

// Erros de regra de negócio dos grupos
var (
	ErrGrupoInexistente = errors.New("Grupo não encontrado")
	ErrSemPermissao     = errors.New("Você não tem permissão para isso")
	ErrGrupoCheio       = errors.New("O grupo atingiu o limite de 67 membros")
	ErrJaEMembro        = errors.New("Esta pessoa já está no grupo")
	ErrGrupoSemMembros  = errors.New("Escolha pelo menos uma pessoa para o grupo")
)

const mensagensPorPagina = 50

// Grupos representa um repositorio de grupos de conversa
type Grupos struct {
	db *sql.DB
}

// NovoRepositorioDeGrupos cria um repositorio de grupos
func NovoRepositorioDeGrupos(db *sql.DB) *Grupos {
	return &Grupos{db: db}
}

// papel devolve se eu sou membro do grupo, se posso adicionar e se sou o criador
func (repositorio Grupos) papel(consulta consultador, eu, grupoID uint64) (membro, podeAdicionar, criador bool, erro error) {
	var pode bool
	var criadorID uint64
	erro = consulta.QueryRow(`
		select m.pode_adicionar, g.criador_id from grupo_membros m
		join grupos g on g.id = m.grupo_id
		where m.grupo_id = ? and m.usuario_id = ?`, grupoID, eu,
	).Scan(&pode, &criadorID)
	if errors.Is(erro, sql.ErrNoRows) {
		return false, false, false, nil
	}
	if erro != nil {
		return false, false, false, erro
	}
	return true, pode || criadorID == eu, criadorID == eu, nil
}

// bloqueados informa se há bloqueio entre duas pessoas, nos dois sentidos (a mesma regra para iniciar uma conversa)
func bloqueados(consulta consultador, a, b uint64) (bool, error) {
	var bloqueio bool
	erro := consulta.QueryRow(`select exists(
		select 1 from bloqueios where (bloqueador_id = ? and bloqueado_id = ?) or (bloqueador_id = ? and bloqueado_id = ?))`,
		a, b, b, a).Scan(&bloqueio)
	return bloqueio, erro
}

func existeUsuario(consulta consultador, id uint64) (bool, error) {
	var existe bool
	erro := consulta.QueryRow("select exists(select 1 from usuarios where id = ?)", id).Scan(&existe)
	return existe, erro
}

func (repositorio Grupos) idsDosMembros(grupoID uint64) []uint64 {
	linhas, erro := repositorio.db.Query("select usuario_id from grupo_membros where grupo_id = ?", grupoID)
	if erro != nil {
		return nil
	}
	defer linhas.Close()
	var ids []uint64
	for linhas.Next() {
		var id uint64
		if linhas.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (repositorio Grupos) avisar(grupoID, exceto uint64) {
	for _, id := range repositorio.idsDosMembros(grupoID) {
		if id != exceto {
			eventos.Publicar(id, eventos.Evento{Tipo: "grupo", De: grupoID})
		}
	}
}

// Criar cria um grupo com o criador e as pessoas escolhidas; ninguém pode estar bloqueado com o criador
func (repositorio Grupos) Criar(eu uint64, nome string, convidados []uint64) (uint64, error) {
	vistos := map[uint64]bool{eu: true}
	var outros []uint64
	for _, id := range convidados {
		if !vistos[id] {
			vistos[id] = true
			outros = append(outros, id)
		}
	}
	if len(outros) == 0 {
		return 0, ErrGrupoSemMembros
	}
	if len(outros)+1 > modelos.MaximoDeMembros {
		return 0, ErrGrupoCheio
	}
	for _, id := range outros {
		existe, erro := existeUsuario(repositorio.db, id)
		if erro != nil {
			return 0, erro
		}
		if !existe {
			return 0, ErrUsuarioInexistente
		}
		bloqueio, erro := bloqueados(repositorio.db, eu, id)
		if erro != nil {
			return 0, erro
		}
		if bloqueio {
			return 0, ErrBloqueado
		}
	}

	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return 0, erro
	}
	defer transacao.Rollback()

	resultado, erro := transacao.Exec("insert into grupos (nome, criador_id) values (?, ?)", nome, eu)
	if erro != nil {
		return 0, erro
	}
	id, erro := resultado.LastInsertId()
	if erro != nil {
		return 0, erro
	}
	grupoID := uint64(id)
	if _, erro = transacao.Exec("insert into grupo_membros (grupo_id, usuario_id, pode_adicionar) values (?, ?, 1)", grupoID, eu); erro != nil {
		return 0, erro
	}
	for _, outro := range outros {
		if _, erro = transacao.Exec("insert into grupo_membros (grupo_id, usuario_id) values (?, ?)", grupoID, outro); erro != nil {
			return 0, erro
		}
	}
	if erro = transacao.Commit(); erro != nil {
		return 0, erro
	}
	repositorio.avisar(grupoID, eu)
	return grupoID, nil
}

// Listar devolve os grupos do usuário, do mais recente para o mais antigo
func (repositorio Grupos) Listar(eu uint64) ([]modelos.Grupo, error) {
	linhas, erro := repositorio.db.Query(`
		select g.id, g.nome, cast(coalesce(unix_timestamp(fg.atualizadoEm), 0) as unsigned),
			(select count(*) from grupo_membros x where x.grupo_id = g.id),
			(select count(*) from grupo_mensagens n where n.grupo_id = g.id and n.id > m.lida_ate and n.remetente_id <> ?),
			coalesce((select n.texto from grupo_mensagens n where n.grupo_id = g.id order by n.id desc limit 1), ''),
			(select n.criadoEm from grupo_mensagens n where n.grupo_id = g.id order by n.id desc limit 1)
		from grupo_membros m join grupos g on g.id = m.grupo_id
		left join fotos_grupos fg on fg.grupo_id = g.id
		where m.usuario_id = ?
		order by coalesce((select max(n.id) from grupo_mensagens n where n.grupo_id = g.id), 0) desc, g.id desc`, eu, eu)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()

	grupos := []modelos.Grupo{}
	for linhas.Next() {
		var g modelos.Grupo
		var ultima sql.NullTime
		if erro = linhas.Scan(&g.ID, &g.Nome, &g.FotoVersao, &g.Membros, &g.NaoLidas, &g.UltimaMensagem, &ultima); erro != nil {
			return nil, erro
		}
		if ultima.Valid {
			g.UltimaEm = &ultima.Time
		}
		grupos = append(grupos, g)
	}
	return grupos, linhas.Err()
}

// Abrir devolve o grupo com membros e mensagens (apos traz só as novas) e marca tudo como lido
func (repositorio Grupos) Abrir(eu, grupoID, apos uint64) (modelos.DialogoDeGrupo, error) {
	var dialogo modelos.DialogoDeGrupo
	membro, pode, criador, erro := repositorio.papel(repositorio.db, eu, grupoID)
	if erro != nil {
		return dialogo, erro
	}
	if !membro {
		return dialogo, ErrGrupoInexistente
	}
	dialogo.ID, dialogo.EuID, dialogo.EuCriador, dialogo.EuPossoAdicionar = grupoID, eu, criador, pode
	if erro = repositorio.db.QueryRow(
		"select g.nome, cast(coalesce(unix_timestamp(fg.atualizadoEm), 0) as unsigned) from grupos g left join fotos_grupos fg on fg.grupo_id = g.id where g.id = ?", grupoID,
	).Scan(&dialogo.Nome, &dialogo.FotoVersao); erro != nil {
		return dialogo, erro
	}

	linhas, erro := repositorio.db.Query(`
		select u.id, u.nome, u.nick, `+colunaFotoVersao+`, g.criador_id = u.id, m.pode_adicionar or g.criador_id = u.id
		from grupo_membros m join grupos g on g.id = m.grupo_id join usuarios u on u.id = m.usuario_id
		left join fotos_usuarios f on f.usuario_id = u.id
		where m.grupo_id = ? order by u.nome`, grupoID)
	if erro != nil {
		return dialogo, erro
	}
	dialogo.ListaDeMembros = []modelos.MembroDeGrupo{}
	for linhas.Next() {
		var m modelos.MembroDeGrupo
		if erro = linhas.Scan(&m.ID, &m.Nome, &m.Nick, &m.FotoVersao, &m.Criador, &m.PodeAdicionar); erro != nil {
			linhas.Close()
			return dialogo, erro
		}
		dialogo.ListaDeMembros = append(dialogo.ListaDeMembros, m)
	}
	linhas.Close()
	dialogo.Membros = uint64(len(dialogo.ListaDeMembros))

	consulta := `
		select * from (
			select n.id, n.remetente_id, u.nome, u.nick, u.admin, n.texto, n.criadoEm
			from grupo_mensagens n join usuarios u on u.id = n.remetente_id
			where n.grupo_id = ? and n.id > ? order by n.id desc limit ?
		) t order by id`
	linhas, erro = repositorio.db.Query(consulta, grupoID, apos, mensagensPorPagina)
	if erro != nil {
		return dialogo, erro
	}
	defer linhas.Close()
	dialogo.MensagensDoGrupo = []modelos.MensagemDeGrupo{}
	var ultimoID uint64
	for linhas.Next() {
		var m modelos.MensagemDeGrupo
		if erro = linhas.Scan(&m.ID, &m.RemetenteID, &m.Nome, &m.Nick, &m.Admin, &m.Texto, &m.CriadoEm); erro != nil {
			return dialogo, erro
		}
		m.Minha = m.RemetenteID == eu
		ultimoID = m.ID
		dialogo.MensagensDoGrupo = append(dialogo.MensagensDoGrupo, m)
	}
	if ultimoID > 0 {
		_, erro = repositorio.db.Exec("update grupo_membros set lida_ate = greatest(lida_ate, ?) where grupo_id = ? and usuario_id = ?", ultimoID, grupoID, eu)
	}
	return dialogo, erro
}

// Enviar publica uma mensagem no grupo e avisa os outros membros
func (repositorio Grupos) Enviar(eu, grupoID uint64, texto string) (modelos.MensagemDeGrupo, error) {
	var mensagem modelos.MensagemDeGrupo
	membro, _, _, erro := repositorio.papel(repositorio.db, eu, grupoID)
	if erro != nil {
		return mensagem, erro
	}
	if !membro {
		return mensagem, ErrGrupoInexistente
	}
	resultado, erro := repositorio.db.Exec("insert into grupo_mensagens (grupo_id, remetente_id, texto) values (?, ?, ?)", grupoID, eu, texto)
	if erro != nil {
		return mensagem, erro
	}
	id, _ := resultado.LastInsertId()
	erro = repositorio.db.QueryRow(
		"select n.id, n.remetente_id, u.nome, u.nick, u.admin, n.texto, n.criadoEm from grupo_mensagens n join usuarios u on u.id = n.remetente_id where n.id = ?", id,
	).Scan(&mensagem.ID, &mensagem.RemetenteID, &mensagem.Nome, &mensagem.Nick, &mensagem.Admin, &mensagem.Texto, &mensagem.CriadoEm)
	if erro != nil {
		return mensagem, erro
	}
	mensagem.Minha = true
	_, _ = repositorio.db.Exec("update grupo_membros set lida_ate = ? where grupo_id = ? and usuario_id = ?", mensagem.ID, grupoID, eu)
	repositorio.avisar(grupoID, eu)
	return mensagem, nil
}

// Adicionar inclui alguém no grupo: só quem tem permissão e sem bloqueio com quem adiciona
func (repositorio Grupos) Adicionar(eu, grupoID, novo uint64) error {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer transacao.Rollback()

	var travado uint64
	if erro = transacao.QueryRow("select id from grupos where id = ? for update", grupoID).Scan(&travado); errors.Is(erro, sql.ErrNoRows) {
		return ErrGrupoInexistente
	} else if erro != nil {
		return erro
	}
	membro, pode, _, erro := repositorio.papel(transacao, eu, grupoID)
	if erro != nil {
		return erro
	}
	if !membro {
		return ErrGrupoInexistente
	}
	if !pode {
		return ErrSemPermissao
	}
	existe, erro := existeUsuario(transacao, novo)
	if erro != nil {
		return erro
	}
	if !existe {
		return ErrUsuarioInexistente
	}
	ja, _, _, erro := repositorio.papel(transacao, novo, grupoID)
	if erro != nil {
		return erro
	}
	if ja {
		return ErrJaEMembro
	}
	var total int
	if erro = transacao.QueryRow("select count(*) from grupo_membros where grupo_id = ?", grupoID).Scan(&total); erro != nil {
		return erro
	}
	if total >= modelos.MaximoDeMembros {
		return ErrGrupoCheio
	}
	bloqueio, erro := bloqueados(transacao, eu, novo)
	if erro != nil {
		return erro
	}
	if bloqueio {
		return ErrBloqueado
	}
	// Quem entra só vê as mensagens a partir de agora
	if _, erro = transacao.Exec(
		"insert into grupo_membros (grupo_id, usuario_id, lida_ate) values (?, ?, coalesce((select max(id) from grupo_mensagens where grupo_id = ?), 0))",
		grupoID, novo, grupoID); erro != nil {
		return erro
	}
	if erro = transacao.Commit(); erro != nil {
		return erro
	}
	repositorio.avisar(grupoID, eu)
	return nil
}

// DefinirPermissao permite ou impede que um membro adicione pessoas; só o criador decide
func (repositorio Grupos) DefinirPermissao(eu, grupoID, alvo uint64, pode bool) error {
	membro, _, criador, erro := repositorio.papel(repositorio.db, eu, grupoID)
	if erro != nil {
		return erro
	}
	if !membro {
		return ErrGrupoInexistente
	}
	if !criador || alvo == eu {
		return ErrSemPermissao
	}
	resultado, erro := repositorio.db.Exec("update grupo_membros set pode_adicionar = ? where grupo_id = ? and usuario_id = ?", pode, grupoID, alvo)
	if erro != nil {
		return erro
	}
	if linhas, _ := resultado.RowsAffected(); linhas == 0 {
		var existe bool
		if erro = repositorio.db.QueryRow("select exists(select 1 from grupo_membros where grupo_id = ? and usuario_id = ?)", grupoID, alvo).Scan(&existe); erro != nil || !existe {
			return ErrUsuarioInexistente
		}
	}
	return nil
}

// Remover tira alguém do grupo: qualquer um sai sozinho e o criador remove os outros.
// Se o criador sai, o posto passa ao membro mais antigo; sem membros o grupo é apagado
func (repositorio Grupos) Remover(eu, grupoID, alvo uint64) error {
	transacao, erro := repositorio.db.Begin()
	if erro != nil {
		return erro
	}
	defer transacao.Rollback()

	var travado uint64
	if erro = transacao.QueryRow("select id from grupos where id = ? for update", grupoID).Scan(&travado); errors.Is(erro, sql.ErrNoRows) {
		return ErrGrupoInexistente
	} else if erro != nil {
		return erro
	}
	membro, _, criador, erro := repositorio.papel(transacao, eu, grupoID)
	if erro != nil {
		return erro
	}
	if !membro {
		return ErrGrupoInexistente
	}
	if alvo != eu && !criador {
		return ErrSemPermissao
	}
	alvoMembro, _, alvoCriador, erro := repositorio.papel(transacao, alvo, grupoID)
	if erro != nil {
		return erro
	}
	if !alvoMembro {
		return ErrUsuarioInexistente
	}

	restantes := repositorio.idsDosMembros(grupoID)
	apagarFoto := false
	if _, erro = transacao.Exec("delete from grupo_membros where grupo_id = ? and usuario_id = ?", grupoID, alvo); erro != nil {
		return erro
	}
	if alvoCriador {
		var novoCriador uint64
		erro = transacao.QueryRow("select usuario_id from grupo_membros where grupo_id = ? order by entrouEm, usuario_id limit 1", grupoID).Scan(&novoCriador)
		switch {
		case errors.Is(erro, sql.ErrNoRows):
			if _, erro = transacao.Exec("delete from grupos where id = ?", grupoID); erro != nil {
				return erro
			}
			apagarFoto = true
		case erro != nil:
			return erro
		default:
			if _, erro = transacao.Exec("update grupos set criador_id = ? where id = ?", novoCriador, grupoID); erro != nil {
				return erro
			}
		}
	}
	if erro = transacao.Commit(); erro != nil {
		return erro
	}
	if apagarFoto {
		midia.Remover(midia.Grupos, grupoID)
	}
	for _, id := range restantes {
		if id != eu {
			eventos.Publicar(id, eventos.Evento{Tipo: "grupo", De: grupoID})
		}
	}
	return nil
}

// SalvarFoto troca a foto do grupo; só criador e organizadores
func (repositorio Grupos) SalvarFoto(eu, grupoID uint64, tipo string, dados []byte) error {
	membro, pode, _, erro := repositorio.papel(repositorio.db, eu, grupoID)
	if erro != nil {
		return erro
	}
	if !membro {
		return ErrGrupoInexistente
	}
	if !pode {
		return ErrSemPermissao
	}
	if erro = midia.Gravar(midia.Grupos, grupoID, dados); erro != nil {
		return erro
	}
	if _, erro = repositorio.db.Exec(
		`insert into fotos_grupos (grupo_id, tipo) values (?, ?)
		on duplicate key update tipo = values(tipo), atualizadoEm = current_timestamp()`, grupoID, tipo); erro != nil {
		return erro
	}
	repositorio.avisar(grupoID, 0)
	return nil
}

// BuscarFoto devolve a foto do grupo para os membros; sql.ErrNoRows quando não há foto
func (repositorio Grupos) BuscarFoto(eu, grupoID uint64) (string, []byte, error) {
	membro, _, _, erro := repositorio.papel(repositorio.db, eu, grupoID)
	if erro != nil {
		return "", nil, erro
	}
	if !membro {
		return "", nil, ErrGrupoInexistente
	}
	var tipo string
	if erro = repositorio.db.QueryRow("select tipo from fotos_grupos where grupo_id = ?", grupoID).Scan(&tipo); erro != nil {
		return "", nil, erro
	}
	dados, erro := lerMidia(midia.Grupos, grupoID)
	return tipo, dados, erro
}

// RemoverFoto tira a foto do grupo; só criador e organizadores
func (repositorio Grupos) RemoverFoto(eu, grupoID uint64) error {
	membro, pode, _, erro := repositorio.papel(repositorio.db, eu, grupoID)
	if erro != nil {
		return erro
	}
	if !membro {
		return ErrGrupoInexistente
	}
	if !pode {
		return ErrSemPermissao
	}
	if _, erro = repositorio.db.Exec("delete from fotos_grupos where grupo_id = ?", grupoID); erro != nil {
		return erro
	}
	repositorio.avisar(grupoID, 0)
	return midia.Remover(midia.Grupos, grupoID)
}

// Candidatos lista quem o usuário segue ou é seguido por ele (e não tem bloqueio), com quem se segue mutuamente primeiro
func (repositorio Grupos) Candidatos(eu uint64, busca string) ([]modelos.PessoaParaGrupo, error) {
	linhas, erro := repositorio.db.Query(`
		select u.id, u.nome, u.nick, `+colunaFotoVersao+`,
			exists(select 1 from seguidores a where a.usuario_id = u.id and a.seguidor_id = ?)
			and exists(select 1 from seguidores b where b.usuario_id = ? and b.seguidor_id = u.id) as mutuo
		from usuarios u left join fotos_usuarios f on f.usuario_id = u.id
		where u.id <> ? and u.email_verificado_em is not null
		and (exists(select 1 from seguidores a where a.usuario_id = u.id and a.seguidor_id = ?)
			or exists(select 1 from seguidores b where b.usuario_id = ? and b.seguidor_id = u.id))
		and not exists(select 1 from bloqueios k where (k.bloqueador_id = ? and k.bloqueado_id = u.id) or (k.bloqueador_id = u.id and k.bloqueado_id = ?))
		and (? = '' or u.nick like concat(?, '%') or u.nome like concat('%', ?, '%'))
		order by mutuo desc, u.nome limit 200`,
		eu, eu, eu, eu, eu, eu, eu, busca, busca, busca)
	if erro != nil {
		return nil, erro
	}
	defer linhas.Close()
	pessoas := []modelos.PessoaParaGrupo{}
	for linhas.Next() {
		var p modelos.PessoaParaGrupo
		if erro = linhas.Scan(&p.ID, &p.Nome, &p.Nick, &p.FotoVersao, &p.Mutuo); erro != nil {
			return nil, erro
		}
		pessoas = append(pessoas, p)
	}
	return pessoas, linhas.Err()
}

// ContarNaoLidas soma as mensagens de grupo que a pessoa ainda não viu
func (repositorio Grupos) ContarNaoLidas(eu uint64) (total uint64, erro error) {
	erro = repositorio.db.QueryRow(`
		select coalesce(sum((
			select count(*) from grupo_mensagens n
			where n.grupo_id = m.grupo_id and n.id > m.lida_ate and n.remetente_id <> ?
		)), 0)
		from grupo_membros m where m.usuario_id = ?`, eu, eu).Scan(&total)
	return
}
