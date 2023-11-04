package performance_test

import (
	p "go-notes/slice/performance"
	"testing"
)

func BenchmarkAppend(b *testing.B)          { p.Append(100000) }
func BenchmarkAppendAllocated(b *testing.B) { p.AppendAllocated(100000) }
func BenchmarkAppendIndexed(b *testing.B)   { p.AppendIndexed(100000) }

func BenchmarkNormalCase(b *testing.B) { p.Normal(1000) }
func BenchmarkBceCase(b *testing.B)    { p.Bce(1000) }
