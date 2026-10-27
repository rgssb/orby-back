package repositorios

import (
	"database/sql"
	"orby/api/src/midia"
	"os"
)

// lerMidia devolve os bytes de um arquivo; sql.ErrNoRows quando ele não existe
func lerMidia(tipoMidia string, id uint64) ([]byte, error) {
	dados, erro := midia.Ler(tipoMidia, id)
	if os.IsNotExist(erro) {
		return nil, sql.ErrNoRows
	}
	return dados, erro
}

// SalvarFoto grava (ou substitui) a foto de perfil do usuario
func (repositorio usuarios) SalvarFoto(usuarioID uint64, tipo string, dados []byte) error {
	if erro := midia.Gravar(midia.Fotos, usuarioID, dados); erro != nil {
		return erro
	}
	_, erro := repositorio.db.Exec(
		`insert into fotos_usuarios (usuario_id, tipo) values (?, ?)
		on duplicate key update tipo = values(tipo), atualizadoEm = current_timestamp()`,
		usuarioID, tipo,
	)
	return erro
}

// BuscarFoto traz a foto de perfil do usuario; sql.ErrNoRows quando ele nao tem foto
func (repositorio usuarios) BuscarFoto(usuarioID uint64) (string, []byte, error) {
	var tipo string
	if erro := repositorio.db.QueryRow(
		"select tipo from fotos_usuarios where usuario_id = ?", usuarioID,
	).Scan(&tipo); erro != nil {
		return "", nil, erro
	}

	dados, erro := lerMidia(midia.Fotos, usuarioID)
	return tipo, dados, erro
}

// RemoverFoto apaga a foto de perfil do usuario
func (repositorio usuarios) RemoverFoto(usuarioID uint64) error {
	if _, erro := repositorio.db.Exec("delete from fotos_usuarios where usuario_id = ?", usuarioID); erro != nil {
		return erro
	}
	return midia.Remover(midia.Fotos, usuarioID)
}

// SalvarBannerImagem grava a imagem própria do banner (e limpa o tema pronto)
func (repositorio usuarios) SalvarBannerImagem(usuarioID uint64, tipo string, dados []byte) error {
	if erro := midia.Gravar(midia.Banners, usuarioID, dados); erro != nil {
		return erro
	}
	_, erro := repositorio.db.Exec(
		`insert into banners_usuarios (usuario_id, preset, tipo) values (?, null, ?)
		on duplicate key update preset = null, tipo = values(tipo), atualizadoEm = current_timestamp()`,
		usuarioID, tipo,
	)
	return erro
}

// SalvarBannerPreset escolhe um tema pronto para o banner (e descarta a imagem própria)
func (repositorio usuarios) SalvarBannerPreset(usuarioID uint64, preset string) error {
	if _, erro := repositorio.db.Exec(
		`insert into banners_usuarios (usuario_id, preset, tipo) values (?, ?, null)
		on duplicate key update preset = values(preset), tipo = null, atualizadoEm = current_timestamp()`,
		usuarioID, preset,
	); erro != nil {
		return erro
	}
	return midia.Remover(midia.Banners, usuarioID)
}

// BuscarBannerImagem traz a imagem do banner; sql.ErrNoRows quando não há
func (repositorio usuarios) BuscarBannerImagem(usuarioID uint64) (string, []byte, error) {
	var tipo string
	if erro := repositorio.db.QueryRow(
		"select tipo from banners_usuarios where usuario_id = ? and tipo is not null", usuarioID,
	).Scan(&tipo); erro != nil {
		return "", nil, erro
	}

	dados, erro := lerMidia(midia.Banners, usuarioID)
	return tipo, dados, erro
}

// RemoverBanner volta o banner ao padrão
func (repositorio usuarios) RemoverBanner(usuarioID uint64) error {
	if _, erro := repositorio.db.Exec("delete from banners_usuarios where usuario_id = ?", usuarioID); erro != nil {
		return erro
	}
	return midia.Remover(midia.Banners, usuarioID)
}
