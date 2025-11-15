package cargo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type CargoToml struct {
	Package struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
	} `toml:"package"`
	Dependencies map[string]interface{} `toml:"dependencies"`
}

func LoadCargoTomlFromDir(dir string) (*CargoToml, error) {
	manifestPath := filepath.Join(dir, "Cargo.toml")

	if _, err := os.Stat(manifestPath); err != nil {
		return nil, fmt.Errorf("cargo toml not found at %s: %w", manifestPath, err)
	}

	cargoToml := CargoToml{}

	if _, err := toml.DecodeFile(manifestPath, &cargoToml); err != nil {
		return nil, fmt.Errorf("failed to decode cargo toml from %s: %w", manifestPath, err)
	}

	return &cargoToml, nil
}

type CargoLock struct {
	Packages []CargoLockPackage `toml:"package"`
}

type CargoLockPackage struct {
	Name         string   `toml:"name"`
	Version      string   `toml:"version"`
	Dependencies []string `toml:"dependencies"`
}

func LoadCargoLockFromDir(dir string) (*CargoLock, error) {
	lockPath := filepath.Join(dir, "Cargo.lock")

	if _, err := os.Stat(lockPath); err != nil {
		return nil, fmt.Errorf("cargo lock not found at %s: %w", lockPath, err)
	}

	lock := CargoLock{}
	if _, err := toml.DecodeFile(lockPath, &lock); err != nil {
		return nil, fmt.Errorf("failed to decode cargo lock from %s: %w", lockPath, err)
	}

	return &lock, nil
}

func BuildAdjacencyFromLock(rootName string, rootDeps []string, lock *CargoLock) map[string][]string {
	adjacency := make(map[string][]string)

	unique := make(map[string]struct{})
	for _, d := range rootDeps {
		name := normalizeDepName(d)
		if name == "" {
			continue
		}
		if _, ok := unique[name]; !ok {
			adjacency[rootName] = append(adjacency[rootName], name)
			unique[name] = struct{}{}
		}
	}

	for _, pkg := range lock.Packages {
		parent := pkg.Name
		for _, dep := range pkg.Dependencies {
			child := normalizeDepName(dep)
			if child == "" {
				continue
			}
			adjacency[parent] = append(adjacency[parent], child)
		}
	}

	return adjacency
}

func normalizeDepName(s string) string {
	if s == "" {
		return ""
	}
	for len(s) > 0 && (s[0] == '"' || s[0] == '\'' || s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last == '"' || last == '\'' || last == ' ' || last == '\t' {
			s = s[:len(s)-1]
			continue
		}
		break
	}
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '(' {
			return s[:i]
		}
	}
	return s
}

func BuildAdjacencyFromTomlRecursively(rootName string, rootDir string, maxDepth int) (map[string][]string, error) {
	type stackItem struct {
		name  string
		dir   string
		depth int
	}
	adj := make(map[string][]string)
	visited := make(map[string]bool)

	stack := []stackItem{{name: rootName, dir: rootDir, depth: 0}}

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if maxDepth > 0 && item.depth > maxDepth {
			continue
		}

		if visited[item.name] {
			continue
		}
		visited[item.name] = true

		tomlData, err := LoadCargoTomlFromDir(item.dir)
		if err != nil {
			continue
		}

		children, localPaths := extractDepsFromToml(tomlData, item.dir)
		adj[item.name] = append(adj[item.name], children...)

		nextDepth := item.depth + 1
		if maxDepth > 0 && nextDepth > maxDepth {
			continue
		}
		for child, p := range localPaths {
			stack = append(stack, stackItem{
				name:  child,
				dir:   p,
				depth: nextDepth,
			})
		}
	}

	return adj, nil
}

func extractDepsFromToml(c *CargoToml, baseDir string) ([]string, map[string]string) {
	children := make([]string, 0, len(c.Dependencies))
	localPaths := make(map[string]string)
	for name, raw := range c.Dependencies {
		children = append(children, name)
		if m, ok := raw.(map[string]interface{}); ok {
			if pv, ok := m["path"]; ok {
				if pathStr, ok := pv.(string); ok && pathStr != "" {
					abs := filepath.Join(baseDir, pathStr)
					localPaths[name] = abs
				}
			}
		}
	}
	return children, localPaths
}
