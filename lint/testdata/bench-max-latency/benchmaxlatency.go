// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package benchmaxlatency

import (
	"testing"
	"time"

	"go.dokimi.dev/assert/bench"
)

func get() {}

func BenchmarkGet(b *testing.B) {
	c := bench.Start(b).MaxLatency(50 * time.Microsecond)
	defer c.End()
	for c.Loop() {
		get()
	}
}
