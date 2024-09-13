package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"strings"

	"github.com/bayraktugrul/modview/internal"

	"github.com/fatih/color"
)

func main() {
	color.Cyan("🚀 Starting modview...")
	logo := `
   __  ___        __     _           
  /  |/  /__  ___/ /  __(_)__ _    __
 / /|_/ / _ \/ _  / |/ / / -_) |/|/ /
/_/  /_/\___/\_,_/|___/_/\__/|__,__/`
	color.Cyan(logo)

	cmd := exec.Command("go", "mod", "graph")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		color.Red("❌ Error running 'go mod graph': %v", err)
		panic(err)
	}

	color.Green("✅ 'go mod graph' command executed successfully.")
	result, err := internal.Convert(strings.NewReader(out.String()))
	if err != nil {
		color.Red("❌ Error converting graph data: %v", err)
		panic(err)
	}

	color.Green("✅ Graph data converted successfully.")

	htmlContent, err := internal.GenerateHTML(result)
	if err != nil {
		color.Red("❌ Error generating HTML: %v", err)
		return
	}
	color.Green("✅ HTML content generated successfully.")

	openInBrowser := flag.Bool("open", false, "Open the temporary file in the default browser")
	flag.Parse()

	if openInBrowser != nil && *openInBrowser {
		color.Green("🔍 Opening HTML content in the default browser...")
		tempFile, err := os.CreateTemp("", "dependency_tree_*.html")
		if err != nil {
			color.Red("❌ Error creating temporary file: %v", err)
			return
		}
		if _, err := tempFile.Write([]byte(htmlContent)); err != nil {
			color.Red("❌ Error writing to temporary file: %v", err)
			return
		}
		if err := internal.OpenInBrowser(tempFile.Name()); err != nil {
			color.Red("❌ Error opening HTML content in the browser: %v", err)
			return
		}
		color.Green("✅ HTML content opened in the default browser.")
		return
	} else {
		if err := os.WriteFile("dependency_tree.html", []byte(htmlContent), 0644); err != nil {
			color.Red("❌ Error writing HTML file: %v", err)
			return
		}
		color.Green("✅ HTML file 'dependency_tree.html' written successfully.")
	}

	color.Cyan("🎉 modview completed successfully.")
}
