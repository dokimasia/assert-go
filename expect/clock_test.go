// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect_test

import (
	"reflect"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
)

// epoch is the instant a controlled clock starts at.
var epoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// controlled keeps the clock that a call of NewControlled returns.
var controlled *expect.Controlled

// TestClock checks the clocks of this surface.
func TestClock(t *testing.T) {
	t.Parallel()

	t.Run("Clock", func(t *testing.T) {
		t.Parallel()

		t.Run("is each clock type of the aborting surface", func(t *testing.T) {
			t.Parallel()
			expect.Equal(t, reflect.TypeFor[expect.Clock](), reflect.TypeFor[assert.Clock](), "one clock")
			expect.Equal(t, reflect.TypeFor[expect.Clocked](), reflect.TypeFor[assert.Clocked](),
				"one clocked seat")
			expect.Equal(t, reflect.TypeFor[expect.System](), reflect.TypeFor[assert.System](),
				"one runtime clock")
			expect.Equal(t, reflect.TypeFor[expect.Controlled](), reflect.TypeFor[assert.Controlled](),
				"one controlled clock")
		})
	})

	t.Run("NewControlled", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a clock that reads the start until it is advanced", func(t *testing.T) {
			t.Parallel()

			c := expect.NewControlled(epoch)
			expect.Equal(t, c.Now(), epoch, "the start")
			c.Advance(time.Hour)
			expect.Equal(t, c.Now(), epoch.Add(time.Hour), "an hour later")
		})
	})
}

// TestClockAllocs checks the allocation ceiling of NewControlled.
func TestClockAllocs(t *testing.T) {
	alloctest.Check(t, clockCases())
}

// BenchmarkClock measures NewControlled.
func BenchmarkClock(b *testing.B) {
	for _, c := range clockCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// clockCases returns a call of NewControlled, with its allocation ceiling,
// measured.
func clockCases() []alloctest.Case {
	return []alloctest.Case{
		{Name: "NewControlled", Call: func(assert.TB) { controlled = expect.NewControlled(epoch) }, Allocs: 2},
	}
}
