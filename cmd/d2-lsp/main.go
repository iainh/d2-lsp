package main

import (
	"log"
	"os"

	"github.com/iainh/d2-lsp/internal/lsp"
)

func main() {
	server := lsp.NewServer()
	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		log.Fatal(err)
	}
}
