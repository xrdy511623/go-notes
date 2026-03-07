package fstree

import (
	"fmt"
	"io"
)

// Component is the unified interface for both files (leaf) and directories
// (composite). Clients that only need Name() and Size() can treat the entire
// tree uniformly — this is the core value of the Composite pattern.
type Component interface {
	Name() string
	Size() int64
}

// ---------------------------------------------------------------------------
// Leaf: File
// ---------------------------------------------------------------------------

// File is a leaf node that has a fixed name and size.
type File struct {
	name string
	size int64
}

// NewFile creates a leaf node.
func NewFile(name string, size int64) *File {
	return &File{name: name, size: size}
}

func (f *File) Name() string { return f.name }
func (f *File) Size() int64  { return f.size }

// ---------------------------------------------------------------------------
// Composite: Directory
// ---------------------------------------------------------------------------

// Directory is a composite node that contains other Components (files or
// sub-directories). It implements Component, so a Directory is itself a
// valid child of another Directory — forming the recursive tree structure.
type Directory struct {
	name     string
	children []Component
}

// NewDirectory creates an empty composite node.
func NewDirectory(name string) *Directory {
	return &Directory{name: name}
}

func (d *Directory) Name() string { return d.name }

// Size returns the total size of all descendants recursively.
// This is the hallmark of Composite: the same operation on a leaf returns a
// scalar, on a composite returns an aggregate, yet the caller uses the same
// interface.
func (d *Directory) Size() int64 {
	var total int64
	for _, child := range d.children {
		total += child.Size()
	}
	return total
}

// Add appends children to this directory. Returns d for chaining.
func (d *Directory) Add(children ...Component) *Directory {
	d.children = append(d.children, children...)
	return d
}

// Children returns the direct children of this directory.
func (d *Directory) Children() []Component {
	return d.children
}

// ---------------------------------------------------------------------------
// Tree operations
// ---------------------------------------------------------------------------

// Walk visits every node in the tree depth-first.
// It uses an interface check for Children() to support any composite type,
// not just *Directory — this is the Go-idiomatic way to handle open recursion.
func Walk(c Component, fn func(Component)) {
	fn(c)
	type container interface{ Children() []Component }
	if dir, ok := c.(container); ok {
		for _, child := range dir.Children() {
			Walk(child, fn)
		}
	}
}

// Search returns all nodes whose name matches (depth-first).
func Search(c Component, name string) []Component {
	var results []Component
	Walk(c, func(node Component) {
		if node.Name() == name {
			results = append(results, node)
		}
	})
	return results
}

// PrintTree writes a tree(1)-style visualization to w.
//
//	project/
//	├── src/
//	│   ├── main.go (1.2 KB)
//	│   └── util.go (300 B)
//	├── README.md (2.0 KB)
//	└── go.mod (150 B)
func PrintTree(w io.Writer, c Component) {
	printNode(w, c, "", "")
}

func printNode(w io.Writer, c Component, prefix, childPrefix string) {
	type container interface{ Children() []Component }
	if dir, ok := c.(container); ok {
		fmt.Fprintf(w, "%s%s/\n", prefix, c.Name())
		children := dir.Children()
		for i, child := range children {
			if i == len(children)-1 {
				printNode(w, child, childPrefix+"└── ", childPrefix+"    ")
			} else {
				printNode(w, child, childPrefix+"├── ", childPrefix+"│   ")
			}
		}
	} else {
		fmt.Fprintf(w, "%s%s (%s)\n", prefix, c.Name(), formatSize(c.Size()))
	}
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
