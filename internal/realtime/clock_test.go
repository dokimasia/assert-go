// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package realtime_test

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/realtime"
)

// monotonic is the field of the text of a time that has a monotonic
// reading.
const monotonic = " m="

// read keeps what a case of the allocation table read, so the compiler
// keeps the call.
var read time.Time

func TestClock(t *testing.T) {
	t.Parallel()

	t.Run("Now", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the instant of the time package with its monotonic reading", func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, realtime.Clock{}.Now().String(), monotonic,
				"outside a bubble the clock returns what time.Now returns")
		})
		t.Run("returns the wall time of the system inside a synctest bubble", func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				got := realtime.Clock{}.Now()
				assert.False(t, strings.Contains(got.String(), monotonic), "the wall time has no monotonic reading")
				assert.True(t, time.Until(got) > 24*time.Hour,
					"the wall time is later than the bubble's clock, which starts at midnight UTC 2000-01-01")
			})
		})
	})
}

// TestClockAllocs checks the allocation ceiling of Now.
func TestClockAllocs(t *testing.T) {
	alloctest.Check(t, clockCases())
}

// BenchmarkClock measures Now.
func BenchmarkClock(b *testing.B) {
	for _, c := range clockCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// clockCases returns a call of Now, with its allocation ceiling.
func clockCases() []alloctest.Case {
	return []alloctest.Case{{Name: "Now", Call: func(assert.TB) { read = realtime.Clock{}.Now() }}}
}
