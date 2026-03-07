package pipeline

import "iter"

// Filter 返回只包含满足条件元素的 iter.Seq[V]。
// 与 slices.Collect + 手动过滤不同，Filter 是惰性的——
// 不满足条件的元素不会触发下游处理。
func Filter[V any](seq iter.Seq[V], pred func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if pred(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Map 对每个元素应用转换函数，返回新类型的 iter.Seq[R]。
// 惰性求值：只有消费方请求时才执行转换。
func Map[V, R any](seq iter.Seq[V], fn func(V) R) iter.Seq[R] {
	return func(yield func(R) bool) {
		for v := range seq {
			if !yield(fn(v)) {
				return
			}
		}
	}
}

// Take 返回前 n 个元素。
func Take[V any](seq iter.Seq[V], n int) iter.Seq[V] {
	return func(yield func(V) bool) {
		count := 0
		for v := range seq {
			if count >= n {
				return
			}
			if !yield(v) {
				return
			}
			count++
		}
	}
}

// TakeWhile 持续产出元素直到 pred 返回 false。
func TakeWhile[V any](seq iter.Seq[V], pred func(V) bool) iter.Seq[V] {
	return func(yield func(V) bool) {
		for v := range seq {
			if !pred(v) {
				return
			}
			if !yield(v) {
				return
			}
		}
	}
}

// Reduce 将 iter.Seq 聚合为单个值。
// 这是唯一非惰性的管道操作——它必须消费完所有元素才能返回结果。
func Reduce[V, R any](seq iter.Seq[V], initial R, fn func(R, V) R) R {
	acc := initial
	for v := range seq {
		acc = fn(acc, v)
	}
	return acc
}

// Concat 将多个 iter.Seq 首尾相连。
func Concat[V any](seqs ...iter.Seq[V]) iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, seq := range seqs {
			for v := range seq {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// Zip 将两个 iter.Seq 按位置配对为 iter.Seq2，以较短的一方为准。
// 内部使用 iter.Pull 将第二个序列从 push 模式转为 pull 模式，
// 从而可以同步推进两个序列。
func Zip[A, B any](a iter.Seq[A], b iter.Seq[B]) iter.Seq2[A, B] {
	return func(yield func(A, B) bool) {
		nextB, stopB := iter.Pull(b)
		defer stopB()
		for va := range a {
			vb, ok := nextB()
			if !ok {
				return
			}
			if !yield(va, vb) {
				return
			}
		}
	}
}
