package repositorios

import "database/sql"

// Silenciados representa um repositorio de usuarios silenciados
type Silenciados struct {
	db *sql.DB
}

// NovoRepositorioDeSilenciados cria um repositorio de silenciados
func NovoRepositorioDeSilenciados(db *sql.DB) *Silenciados {
	return &Silenciados{db: db}
}

// Silenciar esconde as publicações da pessoa dos feeds, sem avisá-la nem mudar o seguimento
func (repositorio Silenciados) Silenciar(silenciadorID, silenciadoID uint64) error {
	_, erro := repositorio.db.Exec("insert ignore into silenciados (silenciador_id, silenciado_id) values (?, ?)", silenciadorID, silenciadoID)
	return erro
}

// DeixarDeSilenciar volta a mostrar as publicações da pessoa nos feeds
func (repositorio Silenciados) DeixarDeSilenciar(silenciadorID, silenciadoID uint64) error {
	_, erro := repositorio.db.Exec("delete from silenciados where silenciador_id = ? and silenciado_id = ?", silenciadorID, silenciadoID)
	return erro
}

// Silenciei diz se o usuário silenciou a outra pessoa
func (repositorio Silenciados) Silenciei(eu, outro uint64) (silenciei bool, erro error) {
	erro = repositorio.db.QueryRow("select exists(select 1 from silenciados where silenciador_id = ? and silenciado_id = ?)", eu, outro).Scan(&silenciei)
	return
}
