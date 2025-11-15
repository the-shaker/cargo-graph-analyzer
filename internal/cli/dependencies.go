package cli

import (
	"fmt"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"cargo-depgraph/internal/cargo"
	"cargo-depgraph/internal/repo"

	"github.com/spf13/cobra"
)

type GetRequest struct {
	Name    string
	Url     string
	Mode    string
	Version string
	Depth   string
}

var getCommand = &cobra.Command{
	Use:   "get",
	Short: "Usage: get [pkg name] [repo url or test file path] [mode (repo/test)] [pkg version] [max depth]",
	Args:  checkArgs,
	RunE:  runGetCommand,
}

func init() {
	rootCommand.AddCommand(getCommand)
}

func checkArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 5 {
		return fmt.Errorf("command must have 5 arguments")
	}

	if args[0] == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	switch args[2] {
	case "repo":
		if !(checkUrl(args[1])) {
			return fmt.Errorf("invalid repo url")
		}
	case "test":
		_, err := os.Stat(args[1])
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file does not exist in this path")
			} else {
				return fmt.Errorf("error with checking file path")
			}
		}
	default:
		return fmt.Errorf("invalid mode. Must be repo or test")
	}

	if args[3] == "" {
		return fmt.Errorf("version cannot be empty")
	}

	_, err := strconv.Atoi(args[4])
	if err != nil {
		return fmt.Errorf("depth must be an integer")
	}

	return nil
}

func runGetCommand(cmd *cobra.Command, args []string) error {
	req := GetRequest{
		Name:    args[0],
		Url:     args[1],
		Mode:    args[2],
		Version: args[3],
		Depth:   args[4],
	}

	dependencies, err := runWithRepoMode(&req)
	if err != nil {
		return err
	}

	fmt.Printf("Dependencies for %s: %s", req.Url, dependencies)

	return nil
}

func runWithRepoMode(req *GetRequest) (string, error) {
	if err := repo.CreateTempDataDirectory(); err != nil {
		return "", err
	}

	defer repo.RemoveTempDataDirectory()

	tempRepoPath, err := repo.CreateTempDataDirectoryForRepo(req.Name)
	if err != nil {
		return "", err
	}

	if err := repo.CloneRepositoryByTag(req.Url, req.Version, req.Name); err != nil {
		return "", err
	}

	cargoToml, err := cargo.LoadCargoTomlFromDir(tempRepoPath)
	if err != nil {
		return "", err
	}

	b, err := json.MarshalIndent(cargoToml.Dependencies, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func checkUrl(url string) bool {
	return (strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "http://") ||
		strings.HasSuffix(url, ".git") ||
		strings.Contains(url, "git@"))
}
