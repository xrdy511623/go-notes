package circularref

// 组合模式的典型陷阱：循环引用导致无限递归
//
// 如果目录 A 包含目录 B，目录 B 又包含目录 A，
// 调用 Size() 或 Walk() 会导致栈溢出（stack overflow）。
//
// 这不是 Go 特有的问题——任何递归树结构都面临此风险。
// 解决方法：① 在 Add 时检测环路  ② 在遍历时维护 visited 集合

type component interface {
	Name() string
	Size() int64
}

type file struct {
	name string
	size int64
}

func (f *file) Name() string { return f.name }
func (f *file) Size() int64  { return f.size }

type directory struct {
	name     string
	children []component
}

func (d *directory) Name() string { return d.name }

// ⚠️ 如果 children 中直接或间接包含 d 本身，这里会无限递归
func (d *directory) Size() int64 {
	var total int64
	for _, child := range d.children {
		total += child.Size()
	}
	return total
}

func (d *directory) Add(c component) { d.children = append(d.children, c) }

// ❌ CircularReference 演示循环引用导致 stack overflow:
//
//	a.Size() → 计算 b.Size() → 计算 a.Size() → 计算 b.Size() → ...
//	fatal error: stack overflow
//
// 运行此函数会导致程序崩溃。
func CircularReference() {
	a := &directory{name: "a"}
	b := &directory{name: "b"}
	a.Add(b)
	b.Add(a) // ⚠️ 形成环路

	_ = a.Size() // fatal: stack overflow
}

// ✅ 正确做法：在 Add 时检测环路
//
//	func (d *Directory) Add(c Component) error {
//	    if containsAncestor(c, d) {
//	        return fmt.Errorf("circular reference: %s already contains %s", c.Name(), d.Name())
//	    }
//	    d.children = append(d.children, c)
//	    return nil
//	}
//
// 或在遍历时维护 visited set：
//
//	func walkSafe(c Component, visited map[Component]bool, fn func(Component)) {
//	    if visited[c] { return }
//	    visited[c] = true
//	    fn(c)
//	    if dir, ok := c.(interface{ Children() []Component }); ok {
//	        for _, child := range dir.Children() {
//	            walkSafe(child, visited, fn)
//	        }
//	    }
//	}
