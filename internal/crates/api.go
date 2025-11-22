package crates

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver/v3"
)

func BuildAdjacencyFromRegistry(rootName string, rootVersion string, maxDepth int) (map[string][]string, error) {
	if rootName == "" || rootVersion == "" {
		return nil, fmt.Errorf("crate name and version must be provided")
	}

	type node struct {
		name    string
		version string
		depth   int
	}

	adjacency := make(map[string][]string)
	visited := make(map[string]bool)
	stack := []node{{name: rootName, version: rootVersion, depth: 0}}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		label := formatNodeLabel(cur.name, cur.version)
		if visited[label] {
			continue
		}
		visited[label] = true

		if maxDepth > 0 && cur.depth > maxDepth {
			continue
		}

		deps, err := fetchDependencies(cur.name, cur.version)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch dependencies for %s@%s: %w", cur.name, cur.version, err)
		}

		children := make([]string, 0, len(deps))
		nextDepth := cur.depth + 1
		for _, dep := range deps {
			if dep.Optional {
				continue
			}
			if dep.Kind != nil && *dep.Kind != "" && *dep.Kind != "normal" {
				continue
			}
			resolvedVersion, err := resolveVersion(dep.CrateID, dep.Req)
			if err != nil {
				continue
			}
			childLabel := formatNodeLabel(dep.CrateID, resolvedVersion)
			children = append(children, childLabel)

			if maxDepth == 0 || nextDepth <= maxDepth {
				stack = append(stack, node{
					name:    dep.CrateID,
					version: resolvedVersion,
					depth:   nextDepth,
				})
			}
		}

		adjacency[label] = children
	}

	return adjacency, nil
}

func formatNodeLabel(name, version string) string {
	return fmt.Sprintf("%s@%s", name, version)
}

type crateDependency struct {
	CrateID  string  `json:"crate_id"`
	Req      string  `json:"req"`
	Optional bool    `json:"optional"`
	Kind     *string `json:"kind"`
}

type dependenciesResponse struct {
	Dependencies []crateDependency `json:"dependencies"`
}

type crateVersion struct {
	Num    string `json:"num"`
	Yanked bool   `json:"yanked"`
}

type versionsResponse struct {
	Versions []crateVersion `json:"versions"`
}

var (
	httpClient = &http.Client{Timeout: 15 * time.Second}

	depCache struct {
		sync.Mutex
		data map[string][]crateDependency
	}

	versionCache struct {
		sync.Mutex
		data map[string][]crateVersion
	}
)

func init() {
	depCache.data = make(map[string][]crateDependency)
	versionCache.data = make(map[string][]crateVersion)
}

func fetchDependencies(name, version string) ([]crateDependency, error) {
	key := formatNodeLabel(name, version)

	depCache.Lock()
	if deps, ok := depCache.data[key]; ok {
		depCache.Unlock()
		return deps, nil
	}
	depCache.Unlock()

	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s/%s/dependencies", name, version)
	body, err := doGET(url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var parsed dependenciesResponse
	if err := json.NewDecoder(body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode dependencies response: %w", err)
	}

	depCache.Lock()
	depCache.data[key] = parsed.Dependencies
	depCache.Unlock()

	return parsed.Dependencies, nil
}

func resolveVersion(name, requirement string) (string, error) {
	req := strings.TrimSpace(requirement)
	var constraint *semver.Constraints
	var err error
	if req != "" {
		constraint, err = semver.NewConstraint(req)
		if err != nil {
			if version, parseErr := semver.NewVersion(req); parseErr == nil {
				return version.Original(), nil
			}
			return "", fmt.Errorf("invalid semver constraint %q for %s: %w", req, name, err)
		}
	}

	versions, err := fetchVersions(name)
	if err != nil {
		return "", err
	}

	var bestVersion *semver.Version
	var bestOriginal string

	for _, v := range versions {
		if v.Yanked {
			continue
		}
		ver, err := semver.NewVersion(v.Num)
		if err != nil {
			continue
		}
		if constraint != nil {
			if !constraint.Check(ver) {
				continue
			}
		}
		if bestVersion == nil || ver.GreaterThan(bestVersion) {
			bestVersion = ver
			bestOriginal = v.Num
		}
	}

	if bestVersion != nil {
		return bestOriginal, nil
	}

	for _, v := range versions {
		if !v.Yanked {
			return v.Num, nil
		}
	}

	return "", fmt.Errorf("no available versions for %s", name)
}

func fetchVersions(name string) ([]crateVersion, error) {
	versionCache.Lock()
	if versions, ok := versionCache.data[name]; ok {
		versionCache.Unlock()
		return versions, nil
	}
	versionCache.Unlock()

	url := fmt.Sprintf("https://crates.io/api/v1/crates/%s/versions", name)
	body, err := doGET(url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var parsed versionsResponse
	if err := json.NewDecoder(body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode versions response: %w", err)
	}

	versionCache.Lock()
	versionCache.data[name] = parsed.Versions
	versionCache.Unlock()

	return parsed.Versions, nil
}

func doGET(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "cargo-graph-analyzer (+https://github.com/shaker)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("crates.io request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return resp.Body, nil
}
