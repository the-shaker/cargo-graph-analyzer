package cli

import (
	"github.com/spf13/cobra"
)

var rootCommand = &cobra.Command{
	Use:   "depgraph",
	Short: "Dependency graph analyzer for Rust (Cargo) packages",
}

func RunCLI() error {
	err := rootCommand.Execute()
	if err != nil {
		return err
	}
	return nil
}