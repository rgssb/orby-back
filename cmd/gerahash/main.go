package main

import (
	"fmt"
	"orby/api/src/seguranca"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: go run ./cmd/gerahash <senha>")
		os.Exit(1)
	}
	hash, _ := seguranca.Hash(os.Args[1])
	fmt.Println(string(hash))
}
