package internal

type Graph struct {
	Root        string
	Edges       []Edge
	MvsPicked   []string
	MvsUnpicked []string
}
