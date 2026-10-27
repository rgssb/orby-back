package main

import (
	"fmt"
	"orby/api/src/seguranca"
)

func main() {
	hash, _ := seguranca.Hash("123456")
	fmt.Println(string(hash))
}
