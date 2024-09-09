# modview

Illuminate your Go project's dependency landscape with modview – a powerful,
interactive visualization tool that transforms the complexity of your module graph into an intuitive,
explorable universe.

modview takes the output of `go mod graph` and weaves it into a dynamic,
browser-based visualization, allowing you to navigate, search, and understand your project's
dependency structure with unprecedented ease. Whether you're optimizing your codebase,
tracking down version conflicts, or simply exploring the ecosystem your project inhabits,
modview provides the map you need to confidently traverse your Go module's dependency terrain.

## Features

- Generates an interactive HTML visualization of your Go module dependencies
- Distinguishes between picked and unpicked dependencies by the Minimal Version Selection (MVS) algorithm
- Allows zooming and panning for easy navigation of large dependency graphs
- Provides a search functionality to quickly find specific dependencies

## Installation

To install modview, use the following command:

```bash
go install github.com/bayraktugrul/modview@latest

Ensure that your Go bin directory is in your system's PATH.

## Usage

Navigate to your Go project's root directory and run:
modview
```

This will generate a file named `dependency_tree.html` in the current directory. Open this file in a web browser to view
your module's dependency graph.

## Visualization Features

- **Zoom Controls**: Use the '+' and '-' buttons in the bottom-left corner or mouse wheel to zoom in and out.
- **Pan**: Click and drag the graph to pan around.
- **Search**: Use the search box in the top-left corner to find specific dependencies.
- **Tooltips**: Hover over truncated dependency names to see the full name.
- **Copy**: Click on a dependency node to copy its full name to the clipboard.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[MIT License](LICENSE)

## Contact

For questions and feedback, please open an issue on the GitHub repository.