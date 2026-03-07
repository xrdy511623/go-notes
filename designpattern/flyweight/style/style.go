package style

import "sync"

// Style 表示不可变的文本样式（内在状态 / intrinsic）。
// 值类型同时用作 map key，所有字段共同构成唯一标识。
type Style struct {
	Font  string
	Size  int
	Color string
	Bold  bool
}

// Registry 是文本样式享元工厂。
// 相同属性组合的样式只创建一次，后续返回同一 *Style 指针。
// 适用于文档渲染：大量文本 span 引用少量共享样式。
type Registry struct {
	mu     sync.RWMutex
	styles map[Style]*Style
}

// NewRegistry 创建一个空的样式注册表。
func NewRegistry() *Registry {
	return &Registry{styles: make(map[Style]*Style)}
}

// Get 返回指定属性组合的共享样式指针。
func (r *Registry) Get(font, color string, size int, bold bool) *Style {
	key := Style{Font: font, Size: size, Color: color, Bold: bold}

	r.mu.RLock()
	if s, ok := r.styles[key]; ok {
		r.mu.RUnlock()
		return s
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.styles[key]; ok {
		return s
	}
	s := &Style{Font: font, Size: size, Color: color, Bold: bold}
	r.styles[key] = s
	return s
}

// Len 返回注册表中唯一样式的数量。
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.styles)
}
