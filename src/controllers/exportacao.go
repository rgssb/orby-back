package controllers

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/midia"
	"orby/api/src/repositorios"
	"orby/api/src/respostas"
	"orby/api/src/seguranca"
	"os"
	"strconv"
	"time"
)

var limiteExportacao = seguranca.NovoLimitador(8, 15*time.Minute)

func ExportarDados(w http.ResponseWriter, r *http.Request) {
	eu, _, ok := parametroAutenticado(w, r, "")
	if !ok {
		return
	}
	var dados struct {
		Senha string `json:"senha"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&dados) != nil || dados.Senha == "" {
		respostas.Erro(w, http.StatusBadRequest, errors.New("Informe sua senha"))
		return
	}
	chave := strconv.FormatUint(eu, 10)
	if limiteExportacao.Bloqueado(chave) {
		respostas.Erro(w, http.StatusTooManyRequests, errors.New("Muitas tentativas. Tente novamente mais tarde"))
		return
	}
	db, err := banco.Conectar()
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer db.Close()
	senha, err := repositorios.NovoRepositorioDeUsuarios(db).BuscarSenha(eu)
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	if seguranca.VerificarSenha(senha, dados.Senha) != nil {
		limiteExportacao.Registrar(chave)
		respostas.Erro(w, http.StatusForbidden, errors.New("Senha incorreta"))
		return
	}
	limiteExportacao.Reiniciar(chave)
	arquivo, err := os.CreateTemp("", "orby-exportacao-*.zip")
	if err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	defer os.Remove(arquivo.Name())
	defer arquivo.Close()
	if err = gerarExportacao(r.Context(), db, arquivo, eu); err != nil {
		respostas.Erro(w, http.StatusInternalServerError, errors.New("Não foi possível gerar a exportação"))
		return
	}
	if _, err = arquivo.Seek(0, io.SeekStart); err != nil {
		respostas.Erro(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="dados-orby.zip"`)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "dados-orby.zip", time.Time{}, arquivo)
}

type consultaExportacao struct {
	nome, consulta string
	args           []interface{}
}

func gerarExportacao(ctx context.Context, db *sql.DB, destino io.Writer, eu uint64) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	zipper := zip.NewWriter(destino)
	consultas := []consultaExportacao{
		{"conta", `select id, nome, nick, email, bio, privado, curtidas_publicas, notas_felas_ativas, criadoEm from usuarios where id = ?`, []interface{}{eu}},
		{"publicacoes", `select p.id, p.titulo, p.conteudo, p.criadoEm, p.editadoEm, r.original_id, (f.publicacao_id is not null) as felas from publicacoes p left join reposts r on r.publicacao_id = p.id left join publicacoes_felas f on f.publicacao_id = p.id where p.autor_id = ? order by p.id`, []interface{}{eu}},
		{"comentarios", `select id, publicacao_id, conteudo, parent_id, criadoEm, editadoEm from comentarios where autor_id = ? order by id`, []interface{}{eu}},
		{"enquetes", `select e.publicacao_id, e.titulo, e.encerra_em from enquetes e join publicacoes p on p.id = e.publicacao_id where p.autor_id = ? order by e.publicacao_id`, []interface{}{eu}},
		{"opcoes_enquetes", `select o.id, o.publicacao_id, o.texto from opcoes_enquetes o join publicacoes p on p.id = o.publicacao_id where p.autor_id = ? order by o.publicacao_id, o.id`, []interface{}{eu}},
		{"meus_votos", `select publicacao_id, opcao_id from votos_enquetes where usuario_id = ? order by publicacao_id`, []interface{}{eu}},
		{"minhas_curtidas", `select publicacao_id from curtidas_publicacoes where usuario_id = ? order by publicacao_id`, []interface{}{eu}},
		{"minhas_reacoes_publicacoes", `select publicacao_id, emoji from reacoes_publicacoes where usuario_id = ? order by publicacao_id`, []interface{}{eu}},
		{"minhas_reacoes_comentarios", `select comentario_id, emoji from reacoes_comentarios where usuario_id = ? order by comentario_id`, []interface{}{eu}},
		{"seguidores", `select seguidor_id from seguidores where usuario_id = ? order by seguidor_id`, []interface{}{eu}},
		{"seguindo", `select usuario_id from seguidores where seguidor_id = ? order by usuario_id`, []interface{}{eu}},
		{"felas", `select membro_id from felas where dono_id = ? order by membro_id`, []interface{}{eu}},
		{"recados", `select id, perfil_id, autor_id, conteudo, criadoEm from notas_felas where perfil_id = ? or autor_id = ? order by id`, []interface{}{eu, eu}},
		{"bloqueados", `select bloqueado_id, criadoEm from bloqueios where bloqueador_id = ? order by bloqueado_id`, []interface{}{eu}},
		{"silenciados", `select silenciado_id, criadoEm from silenciados where silenciador_id = ? order by silenciado_id`, []interface{}{eu}},
		{"salvos", `select publicacao_id, criadoEm from salvos where usuario_id = ? order by publicacao_id`, []interface{}{eu}},
		{"pastas", `select id, nome from pastas_salvos where usuario_id = ? order by id`, []interface{}{eu}},
		{"pastas_publicacoes", `select pasta_id, publicacao_id from pastas_publicacoes where usuario_id = ? order by pasta_id, publicacao_id`, []interface{}{eu}},
		{"preferencias_privacidade", `select mensagens, mencoes, comentarios from preferencias_privacidade where usuario_id = ?`, []interface{}{eu}},
		{"preferencias_notificacoes", `select tipo from notificacoes_silenciadas where usuario_id = ? order by tipo`, []interface{}{eu}},
		{"preferencias_configuracoes", `select tamanho_texto, reduzir_animacoes, pausa_permanente, pausa_ate from preferencias_configuracoes where usuario_id = ?`, []interface{}{eu}},
		{"mensagens", `select m.id, m.conversa_id, m.remetente_id, m.texto, m.responde_a, m.criadoEm from mensagens m join conversas c on c.id = m.conversa_id where (c.usuario_menor = ? or c.usuario_maior = ?) and m.id > if(c.usuario_menor = ?, c.limpa_menor, c.limpa_maior) order by m.id`, []interface{}{eu, eu, eu}},
		{"conversas", `select id, usuario_menor, usuario_maior, criadoEm from conversas where usuario_menor = ? or usuario_maior = ? order by id`, []interface{}{eu, eu}},
		{"mensagens_grupos", `select n.id, n.grupo_id, n.remetente_id, n.texto, n.criadoEm from grupo_mensagens n join grupo_membros m on m.grupo_id = n.grupo_id where m.usuario_id = ? order by n.id`, []interface{}{eu}},
		{"grupos", `select g.id, g.nome, g.criadoEm from grupos g join grupo_membros m on m.grupo_id = g.id where m.usuario_id = ? order by g.id`, []interface{}{eu}},
		{"imagens_publicacoes", `select i.publicacao_id, i.tipo, i.descricao from imagens_publicacoes i join publicacoes p on p.id = i.publicacao_id where p.autor_id = ? order by i.publicacao_id`, []interface{}{eu}},
		{"imagens_extras", `select i.id, i.publicacao_id, i.tipo, i.descricao from imagens_extras_publicacoes i join publicacoes p on p.id = i.publicacao_id where p.autor_id = ? order by i.publicacao_id, i.id`, []interface{}{eu}},
		{"imagens_mensagens", `select i.mensagem_id, i.tipo from imagens_mensagens i join mensagens m on m.id = i.mensagem_id join conversas c on c.id = m.conversa_id where (c.usuario_menor = ? or c.usuario_maior = ?) and m.id > if(c.usuario_menor = ?, c.limpa_menor, c.limpa_maior) order by i.mensagem_id`, []interface{}{eu, eu, eu}},
		{"foto", `select usuario_id, tipo from fotos_usuarios where usuario_id = ?`, []interface{}{eu}},
		{"banner", `select usuario_id, tipo, preset from banners_usuarios where usuario_id = ?`, []interface{}{eu}},
	}
	for _, c := range consultas {
		if err = exportarConsulta(tx, zipper, c); err != nil {
			zipper.Close()
			return err
		}
	}
	midias := []struct {
		tipo, consulta string
		args           []interface{}
	}{
		{midia.Fotos, `select usuario_id, tipo from fotos_usuarios where usuario_id = ?`, []interface{}{eu}},
		{midia.Banners, `select usuario_id, tipo from banners_usuarios where usuario_id = ? and tipo is not null`, []interface{}{eu}},
		{midia.Publicacoes, `select i.publicacao_id, i.tipo from imagens_publicacoes i join publicacoes p on p.id = i.publicacao_id where p.autor_id = ?`, []interface{}{eu}},
		{midia.Extras, `select i.id, i.tipo from imagens_extras_publicacoes i join publicacoes p on p.id = i.publicacao_id where p.autor_id = ?`, []interface{}{eu}},
		{midia.Mensagens, `select i.mensagem_id, i.tipo from imagens_mensagens i join mensagens m on m.id = i.mensagem_id join conversas c on c.id = m.conversa_id where (c.usuario_menor = ? or c.usuario_maior = ?) and m.id > if(c.usuario_menor = ?, c.limpa_menor, c.limpa_maior)`, []interface{}{eu, eu, eu}},
	}
	for _, m := range midias {
		linhas, erro := tx.Query(m.consulta, m.args...)
		if erro != nil {
			zipper.Close()
			return erro
		}
		for linhas.Next() {
			var id uint64
			var tipo string
			if erro = linhas.Scan(&id, &tipo); erro != nil {
				break
			}
			ext, ok := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/gif": ".gif", "image/webp": ".webp"}[tipo]
			if !ok {
				erro = fmt.Errorf("tipo de mídia desconhecido: %s", tipo)
				break
			}
			var conteudo []byte
			conteudo, erro = midia.Ler(m.tipo, id)
			if erro != nil {
				break
			}
			var entrada io.Writer
			entrada, erro = zipper.Create(fmt.Sprintf("midias/%s/%d%s", m.tipo, id, ext))
			if erro == nil {
				_, erro = entrada.Write(conteudo)
			}
			if erro != nil {
				break
			}
		}
		if erro == nil {
			erro = linhas.Err()
		}
		linhas.Close()
		if erro != nil {
			zipper.Close()
			return erro
		}
	}
	if err = zipper.Close(); err != nil {
		return err
	}
	return tx.Commit()
}

func exportarConsulta(tx *sql.Tx, zipper *zip.Writer, c consultaExportacao) error {
	linhas, err := tx.Query(c.consulta, c.args...)
	if err != nil {
		return err
	}
	defer linhas.Close()
	colunas, err := linhas.Columns()
	if err != nil {
		return err
	}
	entrada, err := zipper.Create(c.nome + ".json")
	if err != nil {
		return err
	}
	if _, err = io.WriteString(entrada, "["); err != nil {
		return err
	}
	primeira := true
	for linhas.Next() {
		valores := make([]interface{}, len(colunas))
		destinos := make([]interface{}, len(colunas))
		for i := range valores {
			destinos[i] = &valores[i]
		}
		if err = linhas.Scan(destinos...); err != nil {
			return err
		}
		objeto := make(map[string]interface{}, len(colunas))
		for i, valor := range valores {
			if b, ok := valor.([]byte); ok {
				objeto[colunas[i]] = string(b)
			} else {
				objeto[colunas[i]] = valor
			}
		}
		if !primeira {
			if _, err = io.WriteString(entrada, ","); err != nil {
				return err
			}
		}
		primeira = false
		if err = json.NewEncoder(entrada).Encode(objeto); err != nil {
			return err
		}
	}
	if err = linhas.Err(); err != nil {
		return err
	}
	_, err = io.WriteString(entrada, "]")
	return err
}
