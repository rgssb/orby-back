package main

import (
	"fmt"
	"log"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/config"
	"orby/api/src/router"
	"time"
)

func main() {
	config.Carregar()

	if erro := banco.Migrar(); erro != nil {
		log.Fatalf("erro ao preparar o banco de dados: %v", erro)
	}

	limparMidiaOrfa()
	go func() {
		for range time.Tick(24 * time.Hour) {
			limparMidiaOrfa()
		}
	}()

	r := router.Gerar()

	fmt.Printf("Escutando na porta %d", config.Porta)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Porta), r))

}

func limparMidiaOrfa() {
	db, erro := banco.Conectar()
	if erro != nil {
		log.Printf("limpeza de mídia: %v", erro)
		return
	}
	defer db.Close()

	removidos, erro := banco.LimparMidiaOrfa(db)
	if erro != nil {
		log.Printf("limpeza de mídia: %v", erro)
	}
	if removidos > 0 {
		log.Printf("limpeza de mídia: %d arquivo(s) órfão(s) removido(s)", removidos)
	}
}
