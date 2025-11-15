package graph

import (
	"sort"
	"strings"
)

type AnalysisResult struct {
	Tree          string
	RepeatedNodes []string
	CycleStrings  []string
}

func AnalyzeAndRender(root string, adjacency map[string][]string, maxDepth int) AnalysisResult {
	for k := range adjacency {
		children := adjacency[k]
		sort.Strings(children)
		adjacency[k] = children
	}

	parentSetByNode := make(map[string]map[string]struct{})
	cycleSet := make(map[string]struct{})

	var lines []string

	type frame struct {
		node   string
		depth  int
		prefix string
		children []string
		index    int
		isLast bool
		exit bool
	}

	onPath := make(map[string]bool)
	var currentPath []string

	lines = append(lines, root)

	rootChildren := adjacency[root]
	sort.Strings(rootChildren)
	stack := make([]frame, 0, 64)
	if len(rootChildren) > 0 && maxDepth != 0 {
		for i := len(rootChildren) - 1; i >= 0; i-- {
			isLast := (i == len(rootChildren)-1)
			stack = append(stack, frame{
				node:     rootChildren[i],
				depth:    1,
				prefix:   "",
				children: nil,
				index:    0,
				isLast:   isLast,
				exit:     false,
			})
			if _, ok := parentSetByNode[rootChildren[i]]; !ok {
				parentSetByNode[rootChildren[i]] = make(map[string]struct{})
			}
			parentSetByNode[rootChildren[i]][root] = struct{}{}
		}
	}

	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if !f.exit {
			branch := "|-- "
			pipe := "|   "
			space := "    "
			if f.isLast {
				branch = "`-- "
				pipe = space
			}
			line := f.prefix + branch + f.node + " (depth=" + itoa(f.depth) + ")"
			lines = append(lines, line)

			if onPath[f.node] {
				cycle := extractCycleString(currentPath, f.node)
				if cycle != "" {
					cycleSet[cycle] = struct{}{}
				}
				continue
			}

			stack = append(stack, frame{
				node:     f.node,
				depth:    f.depth,
				prefix:   f.prefix,
				children: nil,
				index:    0,
				isLast:   f.isLast,
				exit:     true,
			})
			onPath[f.node] = true
			currentPath = append(currentPath, f.node)

			if maxDepth > 0 && f.depth >= maxDepth {
				continue
			}

			children := adjacency[f.node]
			if len(children) == 0 {
				continue
			}
			tmp := make([]string, len(children))
			copy(tmp, children)
			sort.Strings(tmp)
			nextPrefix := f.prefix + pipe
			for i := len(tmp) - 1; i >= 0; i-- {
				isLastChild := (i == len(tmp)-1)
				child := tmp[i]
				stack = append(stack, frame{
					node:     child,
					depth:    f.depth + 1,
					prefix:   nextPrefix,
					children: nil,
					index:    0,
					isLast:   isLastChild,
					exit:     false,
				})
				if _, ok := parentSetByNode[child]; !ok {
					parentSetByNode[child] = make(map[string]struct{})
				}
				parentSetByNode[child][f.node] = struct{}{}
			}
		} else {
			if len(currentPath) > 0 && currentPath[len(currentPath)-1] == f.node {
				currentPath = currentPath[:len(currentPath)-1]
			}
			onPath[f.node] = false
		}
	}

	var repeated []string
	for node, parents := range parentSetByNode {
		if len(parents) > 1 {
			repeated = append(repeated, node)
		}
	}
	sort.Strings(repeated)

	var cycles []string
	for c := range cycleSet {
		cycles = append(cycles, c)
	}
	sort.Strings(cycles)

	return AnalysisResult{
		Tree:          strings.Join(lines, "\n"),
		RepeatedNodes: repeated,
		CycleStrings:  cycles,
	}
}

func extractCycleString(path []string, node string) string {
	idx := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == node {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ""
	}
	segment := append(append([]string{}, path[idx:]...), node)
	return strings.Join(segment, " -> ")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		d := byte(n % 10)
		digits = append(digits, '0'+d)
		n /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
