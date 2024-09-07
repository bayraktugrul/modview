package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

type Edge struct {
	From string
	To   string
}

type Graph struct {
	Root        string
	Edges       []Edge
	MvsPicked   []string
	MvsUnpicked []string
}

func Convert(r io.Reader) (*Graph, error) {
	scanner := bufio.NewScanner(r)
	var g Graph
	seen := map[string]bool{}
	mvsPicked := map[string]string{}

	for scanner.Scan() {
		l := scanner.Text()
		if l == "" {
			continue
		}

		parts := strings.Fields(l)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected 2 words in line, but got %d: %s", len(parts), l)
		}

		from := parts[0]
		to := parts[1]
		g.Edges = append(g.Edges, Edge{From: from, To: to})

		for _, node := range []string{from, to} {
			if _, ok := seen[node]; ok {
				continue
			}
			seen[node] = true

			var m, v string
			if i := strings.IndexByte(node, '@'); i >= 0 {
				m, v = node[:i], node[i+1:]
			} else {
				g.Root = node
				continue
			}

			if maxV, ok := mvsPicked[m]; ok {
				if semver.Compare(maxV, v) < 0 {
					g.MvsUnpicked = append(g.MvsUnpicked, m+"@"+maxV)
					mvsPicked[m] = v
				} else {
					g.MvsUnpicked = append(g.MvsUnpicked, node)
				}
			} else {
				mvsPicked[m] = v
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for m, v := range mvsPicked {
		g.MvsPicked = append(g.MvsPicked, m+"@"+v)
	}

	sort.Strings(g.MvsPicked)
	return &g, nil
}

// GenerateHTML generates an HTML representation of the graph using D3.js.
func GenerateHTML(graph *Graph) string {
	// Define the HTML template for the graph.
	const tmpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dependency Graph</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <style>
        body, html {
            margin: 0;
            padding: 0;
            width: 100%;
            height: 100%;
            font-family: Arial, sans-serif;
        }
        #graph-container {
            width: 100%;
            height: 100%;
        }
        .node {
            cursor: pointer;
        }
        .link {
            stroke: #999;
            stroke-opacity: 0.6;
            stroke-width: 2px;
            fill: none;
            marker-end: url(#arrowhead);
        }
        .node text {
            fill: black;
            text-anchor: middle;
            dominant-baseline: middle;
        }
        .tooltip {
            position: absolute;
            background-color: white;
            border: 1px solid #ddd;
            padding: 10px;
            border-radius: 5px;
            pointer-events: none;
        }
    </style>
</head>
<body>
    <div id="graph-container"></div>
    <script>
        const data = {
            nodes: [
                {{- range $node := .Nodes }}
                { id: "{{ $node }}", picked: {{ if in $.MvsPicked $node }}true{{ else if in $.MvsUnpicked $node }}false{{ else }}null{{ end }} },
                {{- end }}
            ],
            links: [
                {{- range $edge := .Edges }}
                { source: "{{ $edge.From }}", target: "{{ $edge.To }}" },
                {{- end }}
            ],
            root: "{{ .Root }}"
        };

        const width = window.innerWidth;
        const height = window.innerHeight;

        const svg = d3.select("#graph-container")
            .append("svg")
            .attr("width", width)
            .attr("height", height);

        // Define arrowhead marker
        svg.append("defs").append("marker")
            .attr("id", "arrowhead")
            .attr("viewBox", "0 -5 10 10")
            .attr("refX", 15)
            .attr("refY", 0)
            .attr("orient", "auto")
            .attr("markerWidth", 8)
            .attr("markerHeight", 8)
            .attr("xoverflow", "visible")
            .append("svg:path")
            .attr("d", "M 0,-5 L 10 ,0 L 0,5")
            .attr("fill", "#999")
            .style("stroke", "none");

        const g = svg.append("g");

        // Create a hierarchical layout
        const hierarchy = d3.stratify()
            .id(d => d.id)
            .parentId(d => {
                const parent = data.links.find(link => link.target === d.id);
                return parent ? parent.source : null;
            })(data.nodes);

        // Calculate the number of nodes and adjust the layout size
        const nodeCount = hierarchy.descendants().length;
        const dynamicWidth = Math.max(width, nodeCount * 100);
        const dynamicHeight = Math.max(height, nodeCount * 50);

        const treeLayout = d3.tree()
            .size([dynamicWidth - 200, dynamicHeight - 200])
            .separation((a, b) => {
                // If the nodes have the same parent and no children, stack them vertically
                if (a.parent === b.parent && (!a.children || a.children.length === 0) && (!b.children || b.children.length === 0)) {
                    return 1;
                }
                return (a.parent == b.parent ? 2 : 3);
            });

        const treeData = treeLayout(hierarchy);

        // Adjust y-coordinates to start from top and ensure minimum vertical spacing
        const minVerticalSpacing = 50;
        treeData.each(d => {
            d.y = height / 6 + d.depth * Math.max(100, minVerticalSpacing);
        });

        // Create a zoom behavior
        const zoom = d3.zoom()
            .scaleExtent([0.1, 2])
            .on("zoom", (event) => {
                g.attr("transform", event.transform);
            });

        svg.call(zoom);

        const link = g.selectAll(".link")
            .data(treeData.links())
            .enter().append("path")
            .attr("class", "link")
            .attr("d", d3.linkVertical()
                .x(d => d.x)
                .y(d => d.y));

        const node = g.selectAll(".node")
            .data(treeData.descendants())
            .enter().append("g")
            .attr("class", "node")
            .attr("transform", function(d) { return "translate(" + d.x + "," + d.y + ")"; });

        // Adjust font size based on node count
        const fontSize = nodeCount > 50 ? 8 : (nodeCount > 20 ? 10 : 12);

        node.append("rect")
            .attr("width", d => Math.min(d.data.id.length * (fontSize * 0.6) + 20, 100))
            .attr("height", 25)
            .attr("x", d => -Math.min(d.data.id.length * (fontSize * 0.6) + 20, 100) / 2)
            .attr("y", -12.5)
            .attr("rx", 5)
            .attr("ry", 5)
            .attr("fill", d => {
                if (d.data.id === data.root) return "#4CAF50";
                if (d.data.picked === true) return "#90EE90";
                if (d.data.picked === false) return "#ccc";
                return "#ccc";
            });

        node.append("text")
            .text(d => d.data.id.length > 15 ? d.data.id.substring(0, 12) + "..." : d.data.id)
            .attr("dy", "0.35em")
            .style("font-size", fontSize + "px");

        // Create tooltip
        const tooltip = d3.select("body").append("div")
            .attr("class", "tooltip")
            .style("opacity", 0);

        // Add mouseover and mouseout events for tooltip
        node.on("mouseover", function(event, d) {
            tooltip.transition()
                .duration(200)
                .style("opacity", .9);
            tooltip.html(d.data.id)
                .style("left", (event.pageX + 10) + "px")
                .style("top", (event.pageY - 28) + "px");
        })
        .on("mouseout", function(d) {
            tooltip.transition()
                .duration(500)
                .style("opacity", 0);
        });

        // Initial centering and scaling
        const rootNode = treeData.descendants()[0];
        const scale = 0.8;
        const initialX = width / 2 - rootNode.x * scale;
        const initialY = height / 6;
        svg.call(zoom.transform, d3.zoomIdentity.translate(initialX, initialY).scale(scale));
    </script>
</body>
</html>
`
	// Define the data to be passed to the template.
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

	// Create a new template and parse the template string into it.
	tmplObj, err := template.New("graph").Funcs(template.FuncMap{
		"in": func(slice []string, item string) bool {
			for _, v := range slice {
				if v == item {
					return true
				}
			}
			return false
		},
	}).Parse(tmpl)
	if err != nil {
		panic(err)
	}

	// Execute the template and write the output to a buffer.
	var buf bytes.Buffer
	if err := tmplObj.Execute(&buf, data); err != nil {
		panic(err)
	}

	// Return the generated HTML as a string.
	return buf.String()
}

// getAllNodes returns a slice of all unique nodes in the graph
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
