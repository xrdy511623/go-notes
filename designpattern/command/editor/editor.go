package editor

import "fmt"

// Command encapsulates a reversible operation on a Document.
// Every concrete command captures enough state at Execute-time to undo itself
// later — this is the core value of the Command pattern.
//
// Standard library parallel: database/sql.Tx batches operations that can be
// rolled back; cobra.Command encapsulates CLI sub-commands as objects.
type Command interface {
	Execute() error
	Undo() error
	String() string
}

// ---------------------------------------------------------------------------
// Receiver: Document
// ---------------------------------------------------------------------------

// Document is the receiver — the actual object being manipulated by commands.
// A real editor would use a rope or piece-table; a simple string suffices
// for demonstrating the pattern.
type Document struct {
	content string
}

func (d *Document) Insert(pos int, text string) error {
	if pos < 0 || pos > len(d.content) {
		return fmt.Errorf("insert position %d out of range [0, %d]", pos, len(d.content))
	}
	d.content = d.content[:pos] + text + d.content[pos:]
	return nil
}

func (d *Document) Delete(pos, length int) (string, error) {
	if pos < 0 || length < 0 || pos+length > len(d.content) {
		return "", fmt.Errorf("delete range [%d, %d) out of bounds [0, %d)", pos, pos+length, len(d.content))
	}
	deleted := d.content[pos : pos+length]
	d.content = d.content[:pos] + d.content[pos+length:]
	return deleted, nil
}

func (d *Document) String() string { return d.content }
func (d *Document) Len() int       { return len(d.content) }

// ---------------------------------------------------------------------------
// Concrete commands
// ---------------------------------------------------------------------------

// InsertCommand inserts text at a position.
// Undo removes exactly the inserted text.
type InsertCommand struct {
	doc  *Document
	pos  int
	text string
}

func NewInsert(doc *Document, pos int, text string) *InsertCommand {
	return &InsertCommand{doc: doc, pos: pos, text: text}
}

func (c *InsertCommand) Execute() error { return c.doc.Insert(c.pos, c.text) }
func (c *InsertCommand) Undo() error    { _, err := c.doc.Delete(c.pos, len(c.text)); return err }
func (c *InsertCommand) String() string { return fmt.Sprintf("Insert(%d, %q)", c.pos, c.text) }

// DeleteCommand deletes `length` bytes at `pos`.
// Execute saves the deleted text; Undo re-inserts it at the same position.
type DeleteCommand struct {
	doc     *Document
	pos     int
	length  int
	deleted string // populated by Execute, consumed by Undo
}

func NewDelete(doc *Document, pos, length int) *DeleteCommand {
	return &DeleteCommand{doc: doc, pos: pos, length: length}
}

func (c *DeleteCommand) Execute() error {
	var err error
	c.deleted, err = c.doc.Delete(c.pos, c.length)
	return err
}

func (c *DeleteCommand) Undo() error    { return c.doc.Insert(c.pos, c.deleted) }
func (c *DeleteCommand) String() string { return fmt.Sprintf("Delete(%d, %d)", c.pos, c.length) }

// ---------------------------------------------------------------------------
// Invoker: Editor
// ---------------------------------------------------------------------------

// Editor is the invoker — it executes commands on a Document and manages the
// undo/redo history stacks. Every Execute pushes onto undo and clears redo;
// Undo pops from undo and pushes onto redo; Redo does the reverse.
type Editor struct {
	doc       *Document
	undoStack []Command
	redoStack []Command
}

// New creates an editor with an empty document.
func New() *Editor {
	return &Editor{doc: &Document{}}
}

// NewWithContent creates an editor with initial content.
func NewWithContent(content string) *Editor {
	return &Editor{doc: &Document{content: content}}
}

// Content returns the current document text.
func (e *Editor) Content() string { return e.doc.String() }

// Doc returns the underlying document (receiver).
func (e *Editor) Doc() *Document { return e.doc }

// Insert is a convenience method that creates and executes an InsertCommand.
func (e *Editor) Insert(pos int, text string) error {
	return e.execute(NewInsert(e.doc, pos, text))
}

// Delete is a convenience method that creates and executes a DeleteCommand.
func (e *Editor) Delete(pos, length int) error {
	return e.execute(NewDelete(e.doc, pos, length))
}

func (e *Editor) execute(cmd Command) error {
	if err := cmd.Execute(); err != nil {
		return err
	}
	e.undoStack = append(e.undoStack, cmd)
	e.redoStack = nil // new action invalidates redo history
	return nil
}

// Undo reverses the last executed command.
func (e *Editor) Undo() error {
	if len(e.undoStack) == 0 {
		return fmt.Errorf("nothing to undo")
	}
	cmd := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	if err := cmd.Undo(); err != nil {
		return err
	}
	e.redoStack = append(e.redoStack, cmd)
	return nil
}

// Redo re-executes the last undone command.
func (e *Editor) Redo() error {
	if len(e.redoStack) == 0 {
		return fmt.Errorf("nothing to redo")
	}
	cmd := e.redoStack[len(e.redoStack)-1]
	e.redoStack = e.redoStack[:len(e.redoStack)-1]
	if err := cmd.Execute(); err != nil {
		return err
	}
	e.undoStack = append(e.undoStack, cmd)
	return nil
}

// CanUndo reports whether there are commands to undo.
func (e *Editor) CanUndo() bool { return len(e.undoStack) > 0 }

// CanRedo reports whether there are commands to redo.
func (e *Editor) CanRedo() bool { return len(e.redoStack) > 0 }

// UndoCount returns the number of undoable commands.
func (e *Editor) UndoCount() int { return len(e.undoStack) }

// RedoCount returns the number of redoable commands.
func (e *Editor) RedoCount() int { return len(e.redoStack) }

// History returns a description of all commands in the undo stack (oldest first).
func (e *Editor) History() []string {
	h := make([]string, len(e.undoStack))
	for i, cmd := range e.undoStack {
		h[i] = cmd.String()
	}
	return h
}
