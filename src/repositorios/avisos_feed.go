package repositorios

import "orby/api/src/eventos"

// AvisarNovaPublicacao avisa em tempo real quem segue o autor (e não o silenciou) que há publicação nova no feed
func (repositorio Publicacoes) AvisarNovaPublicacao(autorID uint64) error {
	return repositorio.avisarNovaPublicacao(autorID, false)
}

func (repositorio Publicacoes) AvisarNovaPublicacaoFelas(autorID uint64) error {
	return repositorio.avisarNovaPublicacao(autorID, true)
}

func (repositorio Publicacoes) avisarNovaPublicacao(autorID uint64, somenteFelas bool) error {
	linhas, erro := repositorio.db.Query(`
		select s.seguidor_id from seguidores s
		where s.usuario_id = ?
		and not exists (select 1 from silenciados m where m.silenciador_id = s.seguidor_id and m.silenciado_id = s.usuario_id)
		and (? = 0 or exists (select 1 from felas fl where fl.dono_id = s.usuario_id and fl.membro_id = s.seguidor_id
			and not exists (select 1 from bloqueios b where
				(b.bloqueador_id = fl.dono_id and b.bloqueado_id = fl.membro_id) or
				(b.bloqueador_id = fl.membro_id and b.bloqueado_id = fl.dono_id))))`,
		autorID, somenteFelas)
	if erro != nil {
		return erro
	}
	defer linhas.Close()

	for linhas.Next() {
		var seguidorID uint64
		if erro = linhas.Scan(&seguidorID); erro != nil {
			return erro
		}
		eventos.Publicar(seguidorID, eventos.Evento{Tipo: "publicacao", De: autorID})
	}
	return linhas.Err()
}
