// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package benchmaxmean

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
)

func get() {}

func BenchmarkGet(b *testing.B) {
	c := bench.Start(b).MaxMean(time.Microsecond)
	defer c.End()
	for c.Loop() {
		get()
	}
}

func BenchmarkPut(b *testing.B) {
	for b.Loop() {
		get()
	}
	if b.Elapsed()/time.Duration(b.N) > time.Microsecond { // want `bench-max-mean: state the check with bench\.Contract\.MaxMean of b\.Elapsed\(\)`
		b.Fatal("Put takes more than a microsecond")
	}
}

func measured(t *testing.T) {
	result := testing.Benchmark(BenchmarkGet)
	assert.True(t, result.NsPerOp() < 1000, "Get takes less than a microsecond") // want `bench-max-mean: state the check with bench.Contract.MaxMean`
}
