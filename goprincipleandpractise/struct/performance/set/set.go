package set

import (
	"strconv"
	"testing"
)

type Set interface {
	Has(string) bool
	Add(string)
	Delete(string)
}

type boolSet map[string]bool

type emptyStructSet map[string]struct{}

func (s emptyStructSet) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s emptyStructSet) Add(key string) {
	s[key] = struct{}{}
}

func (s emptyStructSet) Delete(key string) {
	delete(s, key)
}

func (s boolSet) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s boolSet) Add(key string) {
	s[key] = true
}

func (s boolSet) Delete(key string) {
	delete(s, key)
}

func UseSet(n int, s Set) {
	for i := 0; i < n; i++ {
		s.Add(strconv.Itoa(i))
		if s.Has(strconv.Itoa(i)) {
			s.Delete(strconv.Itoa(i))
		}
	}
}

func Benchmark(b *testing.B, n int, s Set) {
	for i := 0; i < b.N; i++ {
		UseSet(n, s)
	}
}
