// Reference: https://github.com/golang/exp/tree/master/cmd/modgraphviz

package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"sort"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

func Convert(r io.Reader, goModPath string) (*Graph, error) {
	scanner := bufio.NewScanner(r)
	var g Graph
	seen := make(map[string]bool)
	mvsPicked := make(map[string]string)

	rootNode, err := getRootNode(goModPath)
	if err != nil {
		return nil, err
	}
	g.Root = rootNode

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected 2 words in line, but got %d: %s", len(parts), line)
		}

		from, to := parts[0], parts[1]
		if (to != g.Root && !strings.Contains(to, "@")) || (from != g.Root && !strings.Contains(from, "@")) {
			continue
		}

		g.Edges = append(g.Edges, Edge{From: from, To: to})

		for _, node := range []string{from, to} {
			if seen[node] {
				continue
			}
			seen[node] = true

			var module, version string
			if i := strings.IndexByte(node, '@'); i >= 0 {
				module, version = node[:i], node[i+1:]
			}

			if module != g.Root && version == "" {
				continue
			}

			if maxVersion, exists := mvsPicked[module]; exists {
				if semver.Compare(maxVersion, version) < 0 {
					g.MvsUnpicked = append(g.MvsUnpicked, module+"@"+maxVersion)
					mvsPicked[module] = version
				} else {
					g.MvsUnpicked = append(g.MvsUnpicked, node)
				}
			} else {
				mvsPicked[module] = version
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for module, version := range mvsPicked {
		g.MvsPicked = append(g.MvsPicked, module+"@"+version)
	}

	sort.Strings(g.MvsPicked)
	return &g, nil
}

func GenerateHTML(graph *Graph) (string, error) {
	data := struct {
		Nodes       []string
		MvsPicked   []string
		MvsUnpicked []string
		Edges       []Edge
		Root        string
	}{
		Nodes:       getAllNodes(graph),
		MvsPicked:   graph.MvsPicked,
		MvsUnpicked: graph.MvsUnpicked,
		Edges:       graph.Edges,
		Root:        graph.Root,
	}

	tmplObj, err := template.New("dependencyTree").Funcs(template.FuncMap{
		"in": func(slice []string, item string) bool {
			for _, v := range slice {
				if v == item {
					return true
				}
			}
			return false
		},
	}).Parse(Template)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmplObj.Execute(&buf, data); err != nil {
		panic(err)
	}

	return buf.String(), nil
}

func getAllNodes(graph *Graph) []string {
	nodeSet := make(map[string]bool)
	nodeSet[graph.Root] = true
	for _, edge := range graph.Edges {
		nodeSet[edge.From] = true
		nodeSet[edge.To] = true
	}

	nodes := make([]string, 0, len(nodeSet))
	for node := range nodeSet {
		nodes = append(nodes, node)
	}
	return nodes
}

func getRootNode(modPath string) (result string, err error) {
	goModFile, err := os.ReadFile(modPath)
	if err != nil {
		return result, fmt.Errorf("could not read go.mod file: %v", err)
	}

	modFile, err := modfile.Parse("go.mod", goModFile, nil)
	if err != nil {
		return result, fmt.Errorf("could not parse go.mod file: %v", err)
	}

	if modFile.Module == nil {
		return result, fmt.Errorf("go mod is not expected format. module not found")
	}
	return modFile.Module.Mod.Path, nil
}
