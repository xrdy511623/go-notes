package performance

import "testing"

func BenchmarkAppend(b *testing.B)          { Append(100000) }
func BenchmarkAppendAllocated(b *testing.B) { AppendAllocated(100000) }
func BenchmarkAppendIndexed(b *testing.B)   { AppendIndexed(100000) }

func BenchmarkNormalCase(b *testing.B) { Normal(1000) }
func BenchmarkBceCase(b *testing.B)    { Bce(1000) }
