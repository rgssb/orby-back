package main

import (
	"orby/api/src/seguranca"
	"fmt"
)

func main() {
	hash, _ := seguranca.Hash("123456")
	fmt.Println(string(hash))
}
