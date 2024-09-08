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
            fill: none;
            marker-end: url(#arrowhead);
            transition: stroke 0.3s, stroke-width 0.3s;
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
            padding: 5px;
            border-radius: 3px;
            pointer-events: none;
        }
        .highlighted {
            stroke: #0288D1;
            stroke-width: 3px;
        }
        .node-highlighted rect {
            stroke: #0288D1;
            stroke-width: 3px;
        }
        .link-highlighted {
            stroke: #0288D1;
            stroke-width: 3px;
            filter: drop-shadow(0 0 3px #0288D1);
        }
        .node-highlighted-text {
            font-weight: bold;
        }
        .node-highlighted-bg {
            fill: #fff7e6;
        }
        .node-picked-highlight rect {
            stroke: #81D4FA;
            stroke-width: 3px;
        }
        .node-unpicked-highlight rect {
            stroke: #B0BEC5;
            stroke-width: 3px;
        }
        .node-hover rect {
            stroke: #0288D1;
            stroke-width: 3px;
        }
        #legend {
            position: fixed;
            top: 10px;
            left: 10px;
            background-color: rgba(255, 255, 255, 0.8);
            padding: 10px;
            border-radius: 5px;
            box-shadow: 0 0 10px rgba(0,0,0,0.1);
        }
        .legend-item {
            display: flex;
            align-items: center;
            margin-bottom: 5px;
        }
        .legend-color {
            width: 20px;
            height: 20px;
            margin-right: 10px;
            border: 1px solid #000;
        }
    </style>
</head>
<body>
    <div id="legend">
        <div class="legend-item">
            <div class="legend-color" style="background-color: #B2EBF2;"></div>
            <span>Picked dependency by MVS algorithm</span>
        </div>
        <div class="legend-item">
            <div class="legend-color" style="background-color: #CFD8DC;"></div>
            <span>Unpicked dependency</span>
        </div>
    </div>
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
        const nodeCount = data.nodes.length;
        const dynamicWidth = Math.max(width, nodeCount * 80);
        const dynamicHeight = Math.max(height, nodeCount * 40);

        const treeLayout = d3.tree()
            .size([dynamicWidth - 200, dynamicHeight - 200])
            .separation((a, b) => {
                return (a.parent == b.parent ? 1 : 2) / (nodeCount > 50 ? 2 : 1);
            });

        const treeData = treeLayout(hierarchy);

        // Adjust y-coordinates to start from top and ensure minimum vertical spacing
        const minVerticalSpacing = nodeCount > 50 ? 30 : 50;
        treeData.each(d => {
            d.y = height / 6 + d.depth * Math.max(80, minVerticalSpacing);
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
            .attr("d", d3.linkHorizontal()
                .x(d => d.y)
                .y(d => d.x))
            .style("stroke-width", nodeCount > 50 ? "0.5px" : (nodeCount > 20 ? "1px" : "2px"));

        const node = g.selectAll(".node")
            .data(treeData.descendants())
            .enter().append("g")
            .attr("class", "node")
            .attr("transform", d => "translate(" + d.y + "," + d.x + ")");

        // Adjust font size and node size based on node count
        const fontSize = nodeCount <= 15 ? 12 : (nodeCount > 50 ? 6 : (nodeCount > 20 ? 8 : 10));
        const nodeWidth = d => {
            if (nodeCount <= 15) {
                return Math.max(...treeData.descendants().map(n => n.data.id.length * (fontSize * 0.6) + 20));
            }
            return Math.max(Math.min(d.data.id.length * (fontSize * 0.6) + 10, 120), 60);
        };
        const nodeHeight = nodeCount <= 15 ? fontSize * 4 : fontSize * 3;

        node.append("rect")
            .attr("width", d => nodeWidth(d))
            .attr("height", nodeHeight)
            .attr("x", d => -nodeWidth(d) / 2)
            .attr("y", -nodeHeight / 2)
            .attr("rx", 3)
            .attr("ry", 3)
            .attr("fill", d => {
                if (d.data.id === data.root) return "#00BCD4";
                if (d.data.picked === true) return "#B2EBF2";
                if (d.data.picked === false) return "#CFD8DC";
                return "#ccc";
            });

        const nodeText = node.append("text")
            .attr("dy", "0.35em")
            .style("font-size", fontSize + "px");

        // Function to truncate text
        function truncateText(text, maxLength) {
            return text.length > maxLength ? text.slice(0, maxLength - 3) + "..." : text;
        }

        // Create tooltip
        const tooltip = d3.select("body").append("div")
            .attr("class", "tooltip")
            .style("opacity", 0);

        nodeText.each(function(d) {
            const text = d3.select(this);
            const words = d.data.id.split(/(?=[A-Z@])/g);
            const lineHeight = 1.1;
            const y = text.attr("y");
            const dy = parseFloat(text.attr("dy"));
            let tspan = text.text(null).append("tspan").attr("x", 0).attr("y", y).attr("dy", dy + "em");
            let lineNumber = 0;
            let line = [];
            let isTruncated = false;

            words.forEach((word, i) => {
                line.push(word);
                tspan.text(line.join(""));
                if (tspan.node().getComputedTextLength() > nodeWidth(d) - 4) {
                    if (lineNumber === 1) {
                        line.pop();
                        tspan.text(truncateText(line.join(""), line.join("").length - 3));
                        isTruncated = true;
                        return;
                    }
                    line.pop();
                    tspan.text(line.join(""));
                    line = [word];
                    tspan = text.append("tspan").attr("x", 0).attr("y", y).attr("dy", ++lineNumber * lineHeight + dy + "em").text(word);
                }
            });

            if (isTruncated) {
                d3.select(this.parentNode)
                    .on("mouseover", function(event, d) {
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
            }
        });

        // Center text for root node
        node.filter(d => d.data.id === data.root)
            .select("text")
            .attr("text-anchor", "middle")
            .attr("dominant-baseline", "middle")
            .selectAll("tspan")
            .attr("x", 0)
            .attr("dy", (d, i) => i ? "1.1em" : 0);

        // Adjust node positions to prevent overlapping
        const simulation = d3.forceSimulation(treeData.descendants())
            .force("collide", d3.forceCollide().radius(d => nodeWidth(d) / 2 + 10))
            .force("x", d3.forceX(d => d.y).strength(1))
            .force("y", d3.forceY(d => d.x).strength(1))
            .stop();

        for (let i = 0; i < 100; i++) {
            simulation.tick();
        }

        // Update node positions
        node.attr("transform", d => "translate(" + d.y + "," + d.x + ")");

        // Update links
        link.attr("d", d3.linkHorizontal()
            .x(d => d.y)
            .y(d => d.x));

        // Initial centering and scaling
        const rootNode = treeData.descendants()[0];
        const scale = nodeCount <= 15 ? 0.9 : 0.8;
        const initialX = width / 2 - rootNode.y * scale;
        const initialY = height / 6;
        svg.call(zoom.transform, d3.zoomIdentity.translate(initialX, initialY).scale(scale));

        // Highlight path to root on mouseover
        node.on("mouseover", function(event, d) {
            highlightPathToRoot(d);
            highlightNodeAndRoot(d);
            d3.select(this).classed("node-hover", true);
        }).on("mouseout", function() {
            clearHighlight();
            clearNodeAndRootHighlight();
            d3.select(this).classed("node-hover", false);
        }).on("click", function(event, d) {
            focusOnParentNode(d);
        });

        // Highlight edges on mouseover
        link.on("mouseover", function(event, d) {
            d3.select(this).classed("link-highlighted", true);
            highlightConnectedNodes(d);
        }).on("mouseout", function() {
            d3.select(this).classed("link-highlighted", false);
            clearNodeHighlight();
        });

        function highlightPathToRoot(node) {
            let current = node;
            while (current) {
                d3.select(current.node).select("rect").classed("highlighted", true);
                if (current.parent) {
                    const linkToParent = link.filter(l => l.target === current && l.source === current.parent);
                    linkToParent.classed("highlighted", true);
                }
                current = current.parent;
            }
        }

        function clearHighlight() {
            node.select("rect").classed("highlighted", false);
            link.classed("highlighted", false);
        }

        function highlightConnectedNodes(d) {
            d3.select(d.source.node).classed("node-highlighted", true);
            d3.select(d.target.node).classed("node-highlighted", true);
            d3.select(d.source.node).select("rect").classed("node-highlighted-bg", true);
            d3.select(d.target.node).select("rect").classed("node-highlighted-bg", true);
            d3.select(d.source.node).select("text").classed("node-highlighted-text", true);
            d3.select(d.target.node).select("text").classed("node-highlighted-text", true);
        }

        function clearNodeHighlight() {
            node.classed("node-highlighted", false);
            node.select("rect").classed("node-highlighted-bg", false);
            node.select("text").classed("node-highlighted-text", false);
        }

        function focusOnParentNode(d) {
            if (d.parent) {
                const scale = 2; // Increased zoom level
                const x = width / 2 - d.parent.y * scale;
                const y = height / 2 - d.parent.x * scale;

                svg.transition()
                    .duration(750)
                    .call(
                        zoom.transform,
                        d3.zoomIdentity.translate(x, y).scale(scale)
                    );
            }
        }

        function highlightNodeAndRoot(d) {
            const rootNode = treeData.descendants().find(node => node.data.id === data.root);
            const highlightClass = d.data.picked ? "node-picked-highlight" : "node-unpicked-highlight";
            
            d3.select(d.node).classed(highlightClass, true);
            d3.select(rootNode.node).classed(highlightClass, true);
        }

        function clearNodeAndRootHighlight() {
            node.classed("node-picked-highlight", false);
            node.classed("node-unpicked-highlight", false);
        }
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
