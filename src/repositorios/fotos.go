package repositorios

// SalvarFoto grava (ou substitui) a foto de perfil do usuario
func (repositorio usuarios) SalvarFoto(usuarioID uint64, tipo string, dados []byte) error {
	_, erro := repositorio.db.Exec(
		`insert into fotos_usuarios (usuario_id, tipo, dados) values (?, ?, ?)
		on duplicate key update tipo = values(tipo), dados = values(dados), atualizadoEm = current_timestamp()`,
		usuarioID, tipo, dados,
	)
	return erro
}

// BuscarFoto traz a foto de perfil do usuario; sql.ErrNoRows quando ele nao tem foto
func (repositorio usuarios) BuscarFoto(usuarioID uint64) (string, []byte, error) {
	var (
		tipo  string
		dados []byte
	)

	erro := repositorio.db.QueryRow(
		"select tipo, dados from fotos_usuarios where usuario_id = ?", usuarioID,
	).Scan(&tipo, &dados)
	if erro != nil {
		return "", nil, erro
	}

	return tipo, dados, nil
}

// RemoverFoto apaga a foto de perfil do usuario
func (repositorio usuarios) RemoverFoto(usuarioID uint64) error {
	_, erro := repositorio.db.Exec("delete from fotos_usuarios where usuario_id = ?", usuarioID)
	return erro
}

// SalvarBannerImagem grava a imagem própria do banner (e limpa o tema pronto)
func (repositorio usuarios) SalvarBannerImagem(usuarioID uint64, tipo string, dados []byte) error {
	_, erro := repositorio.db.Exec(
		`insert into banners_usuarios (usuario_id, preset, tipo, dados) values (?, null, ?, ?)
		on duplicate key update preset = null, tipo = values(tipo), dados = values(dados), atualizadoEm = current_timestamp()`,
		usuarioID, tipo, dados,
	)
	return erro
}

// SalvarBannerPreset escolhe um tema pronto para o banner (e descarta a imagem própria)
func (repositorio usuarios) SalvarBannerPreset(usuarioID uint64, preset string) error {
	_, erro := repositorio.db.Exec(
		`insert into banners_usuarios (usuario_id, preset, tipo, dados) values (?, ?, null, null)
		on duplicate key update preset = values(preset), tipo = null, dados = null, atualizadoEm = current_timestamp()`,
		usuarioID, preset,
	)
	return erro
}

// BuscarBannerImagem traz a imagem do banner; sql.ErrNoRows quando não há
func (repositorio usuarios) BuscarBannerImagem(usuarioID uint64) (string, []byte, error) {
	var (
		tipo  string
		dados []byte
	)

	erro := repositorio.db.QueryRow(
		"select tipo, dados from banners_usuarios where usuario_id = ? and tipo is not null", usuarioID,
	).Scan(&tipo, &dados)
	return tipo, dados, erro
}

// RemoverBanner volta o banner ao padrão
func (repositorio usuarios) RemoverBanner(usuarioID uint64) error {
	_, erro := repositorio.db.Exec("delete from banners_usuarios where usuario_id = ?", usuarioID)
	return erro
}
