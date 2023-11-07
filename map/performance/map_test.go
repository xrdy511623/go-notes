package performance_test

import (
	p "go-notes/map/performance"
	"testing"
)

func BenchmarkWithoutPreAlloc(b *testing.B) { p.WithoutPreAlloc(10000) }
func BenchmarkWithPreAlloc(b *testing.B)    { p.PreAlloc(10000) }
