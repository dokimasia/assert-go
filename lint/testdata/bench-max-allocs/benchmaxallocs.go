// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package benchmaxallocs

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
)

func get() {}

func BenchmarkGet(b *testing.B) {
	c := bench.Start(b).MaxAllocs(2)
	defer c.End()
	for c.Loop() {
		get()
	}
}

func measured(t *testing.T) {
	result := testing.Benchmark(BenchmarkGet)
	if result.AllocsPerOp() > 2 { // want `bench-max-allocs: state the check with bench.Contract.MaxAllocs`
		t.Fatalf("Get allocates %d times", result.AllocsPerOp())
	}
	assert.True(t, testing.Benchmark(BenchmarkGet).AllocsPerOp() <= 2, "Get allocates at most twice") // want `bench-max-allocs: state the check with bench\.Contract\.MaxAllocs of testing\.Benchmark\(BenchmarkGet\)\.AllocsPerOp\(\)`
}
