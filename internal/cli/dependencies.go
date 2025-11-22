package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"cargo-depgraph/internal/crates"
	"cargo-depgraph/internal/graph"

	"github.com/spf13/cobra"
)

type GetRequest struct {
	Name     string
	Mode     string
	Version  string
	TestFile string
	Depth    int
}

var getCommand = &cobra.Command{
	Use:   "get",
	Short: "Usage: get [pkg name] [mode repo|test] [version or test file] [max depth]",
	Args:  checkArgs,
	RunE:  runGetCommand,
}

func init() {
	rootCommand.AddCommand(getCommand)
}

func checkArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 4 {
		return fmt.Errorf("command must have 4 arguments")
	}
	if args[0] == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	mode := args[1]
	switch mode {
	case "repo":
		if args[2] == "" {
			return fmt.Errorf("version cannot be empty")
		}
	case "test":
		if args[2] == "" {
			return fmt.Errorf("test file path cannot be empty")
		}
		if _, err := os.Stat(args[2]); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("test file does not exist at this path")
			}
			return fmt.Errorf("failed to check test file: %w", err)
		}
	default:
		return fmt.Errorf("invalid mode. Must be repo or test")
	}

	if _, err := strconv.Atoi(args[3]); err != nil {
		return fmt.Errorf("depth must be an integer")
	}

	return nil
}

func runGetCommand(cmd *cobra.Command, args []string) error {
	depth, _ := strconv.Atoi(args[3])
	req := GetRequest{
		Name:  args[0],
		Mode:  args[1],
		Depth: depth,
	}
	switch req.Mode {
	case "repo":
		req.Version = args[2]
	case "test":
		req.TestFile = args[2]
	default:
		return fmt.Errorf("unsupported mode: %s", req.Mode)
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
	adj, err := crates.BuildAdjacencyFromRegistry(req.Name, req.Version, req.Depth)
	if err != nil {
		return "", err
	}

	rootLabel := fmt.Sprintf("%s@%s", req.Name, req.Version)
	result := graph.AnalyzeAndRender(rootLabel, adj, req.Depth)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dependency graph for %s (max-depth %d):\n\n", rootLabel, req.Depth))
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
	appendLoadOrder(&sb, rootLabel, adj)
	return sb.String(), nil
}

func runWithTestMode(req *GetRequest) (string, error) {
	adj, err := parseTestGraphFile(req.TestFile)
	if err != nil {
		return "", err
	}
	if _, ok := adj[req.Name]; !ok {
		adj[req.Name] = []string{}
	}
	result := graph.AnalyzeAndRender(req.Name, adj, req.Depth)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Dependency graph for %s (max-depth %d):\n\n", req.Name, req.Depth))
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
	appendLoadOrder(&sb, req.Name, adj)
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

func appendLoadOrder(sb *strings.Builder, root string, adjacency map[string][]string) {
	order, err := graph.ComputeLoadOrder(root, adjacency)
	if err != nil {
		sb.WriteString(fmt.Sprintf("\nLoad order unavailable: %v\n", err))
		return
	}
	sb.WriteString("\nLoad order:\n")
	for i, node := range order {
		sb.WriteString(fmt.Sprintf(" %d. %s\n", i+1, node))
	}
}
