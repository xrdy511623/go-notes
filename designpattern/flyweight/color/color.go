package color

import "sync"

// Color 表示一个共享的不可变颜色值（内在状态 / intrinsic）。
type Color struct {
	Name       string
	R, G, B, A uint8
}

// Palette 是颜色享元工厂。
// 同名颜色只创建一次，适用于渲染引擎中大量元素引用少量命名颜色的场景。
type Palette struct {
	mu     sync.RWMutex
	colors map[string]*Color
}

// NewPalette 创建一个空的调色板。
func NewPalette() *Palette {
	return &Palette{colors: make(map[string]*Color)}
}

// Get 返回指定名称的共享颜色。首次调用时创建，后续复用。
func (p *Palette) Get(name string, r, g, b, a uint8) *Color {
	p.mu.RLock()
	if c, ok := p.colors[name]; ok {
		p.mu.RUnlock()
		return c
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.colors[name]; ok {
		return c
	}
	c := &Color{Name: name, R: r, G: g, B: b, A: a}
	p.colors[name] = c
	return c
}

// Len 返回调色板中颜色数量。
func (p *Palette) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.colors)
}
