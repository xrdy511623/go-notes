package fstree

import (
	"bytes"
	"strings"
	"testing"
)

func buildTestTree() *Directory {
	// project/
	//   src/
	//     main.go  (1200)
	//     handler.go (800)
	//   README.md  (2048)
	//   go.mod     (150)
	src := NewDirectory("src")
	src.Add(
		NewFile("main.go", 1200),
		NewFile("handler.go", 800),
	)
	root := NewDirectory("project")
	root.Add(src, NewFile("README.md", 2048), NewFile("go.mod", 150))
	return root
}

func TestFile_Size(t *testing.T) {
	f := NewFile("test.go", 42)
	if f.Size() != 42 {
		t.Fatalf("File.Size() = %d, want 42", f.Size())
	}
}

func TestFile_Name(t *testing.T) {
	f := NewFile("test.go", 42)
	if f.Name() != "test.go" {
		t.Fatalf("File.Name() = %q, want %q", f.Name(), "test.go")
	}
}

func TestDirectory_Size_Recursive(t *testing.T) {
	root := buildTestTree()
	want := int64(1200 + 800 + 2048 + 150)
	if got := root.Size(); got != want {
		t.Fatalf("Directory.Size() = %d, want %d", got, want)
	}
}

func TestDirectory_Size_Empty(t *testing.T) {
	d := NewDirectory("empty")
	if d.Size() != 0 {
		t.Fatalf("empty Directory.Size() = %d, want 0", d.Size())
	}
}

func TestDirectory_Size_SubdirectoryOnly(t *testing.T) {
	src := NewDirectory("src")
	src.Add(NewFile("main.go", 1200), NewFile("handler.go", 800))
	if got := src.Size(); got != 2000 {
		t.Fatalf("src.Size() = %d, want 2000", got)
	}
}

func TestComponent_Uniform(t *testing.T) {
	file := NewFile("a.go", 100)
	dir := NewDirectory("pkg")
	dir.Add(NewFile("b.go", 200))

	components := []Component{file, dir}

	sizes := make([]int64, len(components))
	for i, c := range components {
		sizes[i] = c.Size()
	}
	if sizes[0] != 100 || sizes[1] != 200 {
		t.Fatalf("uniform Size() = %v, want [100 200]", sizes)
	}
}

func TestWalk_VisitsAll(t *testing.T) {
	root := buildTestTree()
	var names []string
	Walk(root, func(c Component) {
		names = append(names, c.Name())
	})
	// depth-first: project, src, main.go, handler.go, README.md, go.mod
	want := []string{"project", "src", "main.go", "handler.go", "README.md", "go.mod"}
	if len(names) != len(want) {
		t.Fatalf("Walk visited %d nodes, want %d: %v", len(names), len(want), names)
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("Walk[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestWalk_LeafOnly(t *testing.T) {
	f := NewFile("single.go", 10)
	var count int
	Walk(f, func(_ Component) { count++ })
	if count != 1 {
		t.Fatalf("Walk on leaf visited %d, want 1", count)
	}
}

func TestSearch_Found(t *testing.T) {
	root := buildTestTree()
	results := Search(root, "main.go")
	if len(results) != 1 {
		t.Fatalf("Search found %d, want 1", len(results))
	}
	if results[0].Name() != "main.go" {
		t.Fatalf("Search result = %q, want %q", results[0].Name(), "main.go")
	}
}

func TestSearch_NotFound(t *testing.T) {
	root := buildTestTree()
	results := Search(root, "nonexistent.go")
	if len(results) != 0 {
		t.Fatalf("Search found %d, want 0", len(results))
	}
}

func TestSearch_FindsDirectory(t *testing.T) {
	root := buildTestTree()
	results := Search(root, "src")
	if len(results) != 1 {
		t.Fatalf("Search found %d, want 1", len(results))
	}
}

func TestSearch_DuplicateNames(t *testing.T) {
	root := NewDirectory("root")
	root.Add(NewFile("readme.md", 10))
	sub := NewDirectory("sub")
	sub.Add(NewFile("readme.md", 20))
	root.Add(sub)

	results := Search(root, "readme.md")
	if len(results) != 2 {
		t.Fatalf("Search found %d, want 2 (duplicate names across dirs)", len(results))
	}
}

func TestPrintTree(t *testing.T) {
	root := buildTestTree()
	var buf bytes.Buffer
	PrintTree(&buf, root)
	output := buf.String()

	mustContain := []string{
		"project/",
		"├── src/",
		"│   ├── main.go",
		"│   └── handler.go",
		"├── README.md",
		"└── go.mod",
	}
	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("PrintTree missing %q\nfull output:\n%s", s, output)
		}
	}
}

func TestPrintTree_SingleFile(t *testing.T) {
	f := NewFile("alone.txt", 42)
	var buf bytes.Buffer
	PrintTree(&buf, f)
	if !strings.Contains(buf.String(), "alone.txt (42 B)") {
		t.Fatalf("PrintTree leaf = %q", buf.String())
	}
}

func TestPrintTree_FormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{2621440, "2.5 MB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestAdd_Chaining(t *testing.T) {
	root := NewDirectory("root").
		Add(NewFile("a.go", 10)).
		Add(NewFile("b.go", 20))

	if len(root.Children()) != 2 {
		t.Fatalf("expected 2 children after chaining, got %d", len(root.Children()))
	}
}

func TestDirectory_DeepNesting(t *testing.T) {
	// root/a/b/c/leaf.txt
	root := NewDirectory("root")
	a := NewDirectory("a")
	b := NewDirectory("b")
	c := NewDirectory("c")
	leaf := NewFile("leaf.txt", 99)
	c.Add(leaf)
	b.Add(c)
	a.Add(b)
	root.Add(a)

	if root.Size() != 99 {
		t.Fatalf("deep Size() = %d, want 99", root.Size())
	}

	var depth int
	Walk(root, func(_ Component) { depth++ })
	if depth != 5 {
		t.Fatalf("Walk depth count = %d, want 5", depth)
	}
}
