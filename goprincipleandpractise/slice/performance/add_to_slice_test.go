package performance

import "testing"

func BenchmarkAppend(b *testing.B) {
	b.ReportAllocs()
	result := 0
	for i := 0; i < b.N; i++ {
		result += Append(1)
	}
	if result == 0 {
		b.Fatal("unexpected zero result")
	}
}

func BenchmarkAppendAllocated(b *testing.B) {
	b.ReportAllocs()
	result := 0
	for i := 0; i < b.N; i++ {
		result += AppendAllocated(1)
	}
	if result == 0 {
		b.Fatal("unexpected zero result")
	}
}

func BenchmarkAppendIndexed(b *testing.B) {
	b.ReportAllocs()
	result := 0
	for i := 0; i < b.N; i++ {
		result += AppendIndexed(1)
	}
	if result == 0 {
		b.Fatal("unexpected zero result")
	}
}

func BenchmarkNormalCase(b *testing.B) {
	b.ReportAllocs()
	s := GenerateSlice(1000)
	b.ResetTimer()
	result := 0
	for i := 0; i < b.N; i++ {
		result = SumNormal(s)
	}
	if result == 0 {
		b.Fatal("unexpected zero result")
	}
}

func BenchmarkBceCase(b *testing.B) {
	b.ReportAllocs()
	s := GenerateSlice(1000)
	b.ResetTimer()
	result := 0
	for i := 0; i < b.N; i++ {
		result = SumBce(s)
	}
	if result == 0 {
		b.Fatal("unexpected zero result")
	}
}

func BenchmarkAppendLoop(b *testing.B) {
	b.ReportAllocs()
	result := 0
	for i := 0; i < b.N; i++ {
		result += AppendLoop(100)
	}
	if result == 0 {
		b.Fatal("unexpected zero result")
	}
}

func BenchmarkAppendSpread(b *testing.B) {
	b.ReportAllocs()
	result := 0
	for i := 0; i < b.N; i++ {
		result += AppendSpread(100)
	}
	if result == 0 {
		b.Fatal("unexpected zero result")
	}
}
