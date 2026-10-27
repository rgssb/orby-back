package main

import (
	"fmt"
	"log"
	"net/http"
	"orby/api/src/banco"
	"orby/api/src/config"
	"orby/api/src/router"
)

func main() {
	config.Carregar()

	if erro := banco.Migrar(); erro != nil {
		log.Fatalf("erro ao preparar o banco de dados: %v", erro)
	}

	r := router.Gerar()

	fmt.Printf("Escutando na porta %d", config.Porta)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Porta), r))

}
