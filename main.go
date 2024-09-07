package main

import (
	"bytes"
	"fmt"
	"github.com/bayraktugrul/modview/internal"
	"os"
	"os/exec"
	"strings"
)

func main() {
	cmd := exec.Command("go", "mod", "graph")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
	graphOutput := out.String()

	reader := strings.NewReader(graphOutput)
	result, err := internal.Convert(reader)
	if err != nil {
		panic(err)
	}
	htmlContent := internal.GenerateHTML(result)
	if err != nil {
		fmt.Println("Error generating HTML:", err)
		return
	}

	err = os.WriteFile("dependency_tree.html", []byte(htmlContent), 0644)
	if err != nil {
		fmt.Println("Error writing HTML file:", err)
	}
}
