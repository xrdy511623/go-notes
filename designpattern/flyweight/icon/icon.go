package icon

import "sync"

// Icon 表示一个共享的图标（内在状态 / intrinsic）。
// 位置、缩放等外在状态（extrinsic）由调用方在渲染时传入，不存储在此处。
type Icon struct {
	Name string
	Data []byte // 位图数据，共享后不可修改
}

// Factory 是图标享元工厂。
// 相同名称的图标只加载一次，后续请求返回同一 *Icon 指针。
type Factory struct {
	mu    sync.RWMutex
	icons map[string]*Icon
}

// NewFactory 创建一个空的图标工厂。
func NewFactory() *Factory {
	return &Factory{icons: make(map[string]*Icon)}
}

// Get 返回指定名称的共享图标。
// 首次访问时通过 loader 加载数据；后续直接复用已缓存的实例。
func (f *Factory) Get(name string, loader func(string) []byte) *Icon {
	f.mu.RLock()
	if icon, ok := f.icons[name]; ok {
		f.mu.RUnlock()
		return icon
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()
	if icon, ok := f.icons[name]; ok {
		return icon
	}
	icon := &Icon{Name: name, Data: loader(name)}
	f.icons[name] = icon
	return icon
}

// Len 返回工厂中缓存的图标数量。
func (f *Factory) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.icons)
}
