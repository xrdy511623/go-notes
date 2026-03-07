package sequence

import (
	"cmp"
	"iter"
	"slices"
)

// OrderedSet 是一个有序集合，元素按自然顺序排列且不重复。
// 它展示了如何为自定义集合实现 iter.Seq / iter.Seq2 迭代器，
// 使集合可以直接用于 for-range 语句（Go 1.23+ range-over-func）。
//
// 四个迭代方法返回的都是 iter.Seq / iter.Seq2 —— 一个接受 yield 回调的函数。
// 调用方用 for-range 消费，编译器负责将 range 翻译为 yield 调用。
// 消费方 break 时 yield 返回 false，生产方立即停止，无 goroutine、无泄漏。
type OrderedSet[E cmp.Ordered] struct {
	items []E
}

// New 创建一个 OrderedSet，元素自动排序去重。
func New[E cmp.Ordered](items ...E) *OrderedSet[E] {
	s := &OrderedSet[E]{}
	for _, item := range items {
		s.Add(item)
	}
	return s
}

// Add 向集合中添加元素，保持有序且去重。
func (s *OrderedSet[E]) Add(item E) {
	pos, found := slices.BinarySearch(s.items, item)
	if found {
		return
	}
	s.items = slices.Insert(s.items, pos, item)
}

// Len 返回集合元素个数。
func (s *OrderedSet[E]) Len() int { return len(s.items) }

// All 返回遍历所有元素的 iter.Seq[E]。
// 调用方可以用 for v := range set.All() 遍历，也可以随时 break。
func (s *OrderedSet[E]) All() iter.Seq[E] {
	return func(yield func(E) bool) {
		for _, item := range s.items {
			if !yield(item) {
				return
			}
		}
	}
}

// Backward 返回逆序遍历的 iter.Seq[E]。
func (s *OrderedSet[E]) Backward() iter.Seq[E] {
	return func(yield func(E) bool) {
		for i := len(s.items) - 1; i >= 0; i-- {
			if !yield(s.items[i]) {
				return
			}
		}
	}
}

// Pairs 返回 (index, element) 的 iter.Seq2[int, E]。
// 与 slices.All() 风格一致。
func (s *OrderedSet[E]) Pairs() iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i, item := range s.items {
			if !yield(i, item) {
				return
			}
		}
	}
}

// Range 返回 [from, to) 区间内元素的 iter.Seq[E]。
// 利用有序性跳过不在区间内的元素。
func (s *OrderedSet[E]) Range(from, to E) iter.Seq[E] {
	return func(yield func(E) bool) {
		for _, item := range s.items {
			if item < from {
				continue
			}
			if item >= to {
				return
			}
			if !yield(item) {
				return
			}
		}
	}
}
