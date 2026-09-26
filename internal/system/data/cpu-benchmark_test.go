package data

import "testing"

func BenchmarkCalculateCPUUsage(b *testing.B) {
	first := [8]uint64{
		100000,
		2000,
		50000,
		800000,
		10000,
		100,
		500,
		50,
	}

	second := [8]uint64{
		100100,
		2010,
		50050,
		800500,
		10020,
		105,
		510,
		52,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		calculateCPUUsage(first, second)
	}
}

func BenchmarkReadCPUStats(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		readCPUStats()
	}
}
