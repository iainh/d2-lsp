package main

import (
	"errors"
	"log"
	"os"

	"github.com/iainh/d2-lsp/internal/lsp"
)

func main() {
	server := lsp.NewServer()
	if err := server.Serve(os.Stdin, os.Stdout); err != nil {
		if errors.Is(err, lsp.ErrExitWithoutShutdown) {
			os.Exit(1)
		}
		log.Fatal(err)
	}
}
