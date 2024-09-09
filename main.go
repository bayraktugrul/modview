package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bayraktugrul/modview/internal"
)

func main() {
	cmd := exec.Command("go", "mod", "graph")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		panic(err)
	}

	result, err := internal.Convert(strings.NewReader(out.String()))
	if err != nil {
		panic(err)
	}

	htmlContent := internal.GenerateHTML(result)
	if err != nil {
		fmt.Println("Error generating HTML:", err)
		return
	}

	if err := os.WriteFile("dependency_tree.html", []byte(htmlContent), 0644); err != nil {
		fmt.Println("Error writing HTML file:", err)
	}
}
