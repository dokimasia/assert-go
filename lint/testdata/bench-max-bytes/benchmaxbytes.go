// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package benchmaxbytes

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
)

func get() {}

func BenchmarkGet(b *testing.B) {
	c := bench.Start(b).MaxBytes(64)
	defer c.End()
	for c.Loop() {
		get()
	}
}

func measured(t *testing.T) {
	result := testing.Benchmark(BenchmarkGet)
	assert.InRange(t, result.AllocedBytesPerOp(), 0, 64, "Get allocates at most 64 bytes") // want `bench-max-bytes: state the check with bench\.Contract\.MaxBytes of result\.AllocedBytesPerOp\(\)`
}
