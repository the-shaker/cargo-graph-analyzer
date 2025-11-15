package main

import (
	"cargo-depgraph/internal/cli"
	"log"
	"os"
)

func main() {
	err := cli.RunCLI()
	if err != nil {
		log.Printf("error with running cli: %v", err)
		os.Exit(1)
	}
}
