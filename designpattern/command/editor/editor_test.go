package editor

import "testing"

func TestEditor_Insert(t *testing.T) {
	ed := New()
	ed.Insert(0, "Hello")
	if ed.Content() != "Hello" {
		t.Fatalf("content = %q, want %q", ed.Content(), "Hello")
	}
}

func TestEditor_InsertAppend(t *testing.T) {
	ed := NewWithContent("Hello")
	ed.Insert(5, " World")
	if ed.Content() != "Hello World" {
		t.Fatalf("content = %q, want %q", ed.Content(), "Hello World")
	}
}

func TestEditor_InsertMiddle(t *testing.T) {
	ed := NewWithContent("Hllo")
	ed.Insert(1, "e")
	if ed.Content() != "Hello" {
		t.Fatalf("content = %q, want %q", ed.Content(), "Hello")
	}
}

func TestEditor_Delete(t *testing.T) {
	ed := NewWithContent("Hello World")
	ed.Delete(5, 6)
	if ed.Content() != "Hello" {
		t.Fatalf("content = %q, want %q", ed.Content(), "Hello")
	}
}

func TestEditor_InsertOutOfRange(t *testing.T) {
	ed := NewWithContent("Hi")
	if err := ed.Insert(10, "X"); err == nil {
		t.Fatal("expected error for out-of-range insert")
	}
}

func TestEditor_DeleteOutOfRange(t *testing.T) {
	ed := NewWithContent("Hi")
	if err := ed.Delete(0, 10); err == nil {
		t.Fatal("expected error for out-of-range delete")
	}
}

func TestEditor_Undo(t *testing.T) {
	ed := New()
	ed.Insert(0, "Hello")
	ed.Undo()
	if ed.Content() != "" {
		t.Fatalf("after undo: content = %q, want %q", ed.Content(), "")
	}
}

func TestEditor_UndoDelete(t *testing.T) {
	ed := NewWithContent("Hello World")
	ed.Delete(5, 6)
	ed.Undo()
	if ed.Content() != "Hello World" {
		t.Fatalf("after undo delete: content = %q, want %q", ed.Content(), "Hello World")
	}
}

func TestEditor_Redo(t *testing.T) {
	ed := New()
	ed.Insert(0, "Hello")
	ed.Undo()
	ed.Redo()
	if ed.Content() != "Hello" {
		t.Fatalf("after redo: content = %q, want %q", ed.Content(), "Hello")
	}
}

func TestEditor_NewAction_ClearsRedo(t *testing.T) {
	ed := New()
	ed.Insert(0, "A")
	ed.Undo()

	if !ed.CanRedo() {
		t.Fatal("should be able to redo after undo")
	}

	ed.Insert(0, "B")
	if ed.CanRedo() {
		t.Fatal("new action should clear redo stack")
	}
}

func TestEditor_MultipleUndoRedo(t *testing.T) {
	ed := New()
	ed.Insert(0, "Hello")  // "Hello"
	ed.Insert(5, " World") // "Hello World"
	ed.Delete(0, 6)        // "World"

	ed.Undo() // "Hello World"
	if ed.Content() != "Hello World" {
		t.Fatalf("undo 1: %q", ed.Content())
	}

	ed.Undo() // "Hello"
	if ed.Content() != "Hello" {
		t.Fatalf("undo 2: %q", ed.Content())
	}

	ed.Undo() // ""
	if ed.Content() != "" {
		t.Fatalf("undo 3: %q", ed.Content())
	}

	ed.Redo() // "Hello"
	if ed.Content() != "Hello" {
		t.Fatalf("redo 1: %q", ed.Content())
	}

	ed.Redo() // "Hello World"
	if ed.Content() != "Hello World" {
		t.Fatalf("redo 2: %q", ed.Content())
	}

	ed.Redo() // "World"
	if ed.Content() != "World" {
		t.Fatalf("redo 3: %q", ed.Content())
	}
}

func TestEditor_UndoEmpty(t *testing.T) {
	ed := New()
	if err := ed.Undo(); err == nil {
		t.Fatal("expected error for undo on empty stack")
	}
}

func TestEditor_RedoEmpty(t *testing.T) {
	ed := New()
	if err := ed.Redo(); err == nil {
		t.Fatal("expected error for redo on empty stack")
	}
}

func TestEditor_CanUndoRedo(t *testing.T) {
	ed := New()
	if ed.CanUndo() {
		t.Fatal("new editor should not have undo")
	}
	if ed.CanRedo() {
		t.Fatal("new editor should not have redo")
	}

	ed.Insert(0, "X")
	if !ed.CanUndo() {
		t.Fatal("should have undo after insert")
	}

	ed.Undo()
	if !ed.CanRedo() {
		t.Fatal("should have redo after undo")
	}
}

func TestEditor_History(t *testing.T) {
	ed := New()
	ed.Insert(0, "Hello")
	ed.Insert(5, " World")
	ed.Delete(0, 6)

	h := ed.History()
	if len(h) != 3 {
		t.Fatalf("history len = %d, want 3", len(h))
	}
	if h[0] != `Insert(0, "Hello")` {
		t.Errorf("history[0] = %q", h[0])
	}
}

func TestEditor_UndoRedoCounts(t *testing.T) {
	ed := New()
	ed.Insert(0, "A")
	ed.Insert(1, "B")

	if ed.UndoCount() != 2 {
		t.Fatalf("undo count = %d, want 2", ed.UndoCount())
	}
	if ed.RedoCount() != 0 {
		t.Fatalf("redo count = %d, want 0", ed.RedoCount())
	}

	ed.Undo()
	if ed.UndoCount() != 1 || ed.RedoCount() != 1 {
		t.Fatalf("after 1 undo: undo=%d redo=%d", ed.UndoCount(), ed.RedoCount())
	}
}

func TestDocument_Operations(t *testing.T) {
	d := &Document{}
	d.Insert(0, "Hello")
	if d.Len() != 5 {
		t.Fatalf("Len = %d, want 5", d.Len())
	}

	deleted, _ := d.Delete(0, 5)
	if deleted != "Hello" {
		t.Fatalf("Delete returned %q, want %q", deleted, "Hello")
	}
	if d.Len() != 0 {
		t.Fatalf("Len after delete = %d, want 0", d.Len())
	}
}
