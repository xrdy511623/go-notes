package undoleak

import "fmt"

// Command 接口（简化版）
type Command interface {
	Execute()
	Undo()
}

// ❌ BadEditor 将每一条命令无限追加到 undoStack。
// 在长时间运行的应用中（如文本编辑器、绘图工具），用户可能执行
// 数万次操作，每条命令都保存了被删除文本/图形的副本 → 内存持续增长，
// 直到 OOM。
type BadEditor struct {
	undoStack []Command // 无界增长，永远不释放
}

func (e *BadEditor) Execute(cmd Command) {
	cmd.Execute()
	e.undoStack = append(e.undoStack, cmd)
}

func (e *BadEditor) Undo() {
	if len(e.undoStack) == 0 {
		return
	}
	cmd := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	cmd.Undo()
}

// ✅ BoundedEditor 限制 undo 栈深度，避免内存泄漏。
// 超出限制时丢弃最老的命令 — 大多数用户不需要撤销到一小时前的操作。
type BoundedEditor struct {
	undoStack  []Command
	maxHistory int
}

func NewBoundedEditor(maxHistory int) *BoundedEditor {
	return &BoundedEditor{maxHistory: maxHistory}
}

func (e *BoundedEditor) Execute(cmd Command) {
	cmd.Execute()
	e.undoStack = append(e.undoStack, cmd)
	if len(e.undoStack) > e.maxHistory {
		// 丢弃最老的命令，释放其持有的数据
		e.undoStack = e.undoStack[len(e.undoStack)-e.maxHistory:]
	}
}

func (e *BoundedEditor) Undo() {
	if len(e.undoStack) == 0 {
		return
	}
	cmd := e.undoStack[len(e.undoStack)-1]
	e.undoStack = e.undoStack[:len(e.undoStack)-1]
	cmd.Undo()
}

func Example() {
	fmt.Println("❌ BadEditor: undoStack 无限增长 → 内存泄漏")
	fmt.Println("   在长时间编辑场景下 (如 IDE、绘图工具)，每个命令")
	fmt.Println("   持有被删除内容的副本，内存只增不减。")
	fmt.Println()
	fmt.Println("✅ BoundedEditor: 限制 maxHistory=100")
	fmt.Println("   超出时丢弃最老命令，内存使用保持稳定。")
	fmt.Println("   实际应用中也可以使用环形缓冲区 (ring buffer)。")
}
