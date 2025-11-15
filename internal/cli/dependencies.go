package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"cargo-depgraph/internal/cargo"
	"cargo-depgraph/internal/graph"
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

	var out string
	var err error
	switch req.Mode {
	case "repo":
		out, err = runWithRepoMode(&req)
	case "test":
		out, err = runWithTestMode(&req)
	default:
		err = fmt.Errorf("unsupported mode: %s", req.Mode)
	}
	if err != nil {
		return err
	}

	fmt.Print(out)

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

	maxDepth, _ := strconv.Atoi(req.Depth)
	adj, err := cargo.BuildAdjacencyFromTomlRecursively(req.Name, tempRepoPath, maxDepth)
	if err != nil {
		return "", err
	}
	result := graph.AnalyzeAndRender(req.Name, adj, maxDepth)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dependency graph for %s (max-depth %d):\n\n", req.Name, maxDepth))
	sb.WriteString(result.Tree)
	sb.WriteString("\n\nRepeated nodes:\n")
	if len(result.RepeatedNodes) == 0 {
		sb.WriteString(" - none\n")
	} else {
		for _, rn := range result.RepeatedNodes {
			sb.WriteString(" - " + rn + "\n")
		}
	}
	if len(result.CycleStrings) > 0 {
		sb.WriteString("\nCycles:\n")
		for _, cs := range result.CycleStrings {
			sb.WriteString(" - " + cs + "\n")
		}
	}
	return sb.String(), nil
}

func checkUrl(url string) bool {
	return (strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "http://") ||
		strings.HasSuffix(url, ".git") ||
		strings.Contains(url, "git@"))
}

func runWithTestMode(req *GetRequest) (string, error) {
	adj, err := parseTestGraphFile(req.Url)
	if err != nil {
		return "", err
	}
	if _, ok := adj[req.Name]; !ok {
		adj[req.Name] = []string{}
	}
	maxDepth, _ := strconv.Atoi(req.Depth)
	result := graph.AnalyzeAndRender(req.Name, adj, maxDepth)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dependency graph for %s (max-depth %d):\n\n", req.Name, maxDepth))
	sb.WriteString(result.Tree)
	sb.WriteString("\n\nRepeated nodes:\n")
	if len(result.RepeatedNodes) == 0 {
		sb.WriteString(" - none\n")
	} else {
		for _, rn := range result.RepeatedNodes {
			sb.WriteString(" - " + rn + "\n")
		}
	}
	if len(result.CycleStrings) > 0 {
		sb.WriteString("\nCycles:\n")
		for _, cs := range result.CycleStrings {
			sb.WriteString(" - " + cs + "\n")
		}
	}
	return sb.String(), nil
}

func parseTestGraphFile(path string) (map[string][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test graph file: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	adj := make(map[string][]string)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		line = strings.ReplaceAll(line, "->", ":")
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			// tolerate standalone node
			name := strings.TrimSpace(line)
			if name != "" {
				if _, ok := adj[name]; !ok {
					adj[name] = []string{}
				}
			}
			continue
		}
		parent := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		right = strings.ReplaceAll(right, ",", " ")
		fields := strings.Fields(right)
		children := make([]string, 0, len(fields))
		for _, f := range fields {
			if f == "" {
				continue
			}
			children = append(children, f)
		}
		adj[parent] = append(adj[parent], children...)
	}
	return adj, nil
}
