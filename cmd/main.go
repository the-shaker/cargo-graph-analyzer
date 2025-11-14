package main

import (
	"cargo-depgraph/internal/cli"
	"log"
)

func main() {
	err := cli.RunCLI()
	if err != nil {
		log.Printf("error with running cli: %v", err)
	}
}