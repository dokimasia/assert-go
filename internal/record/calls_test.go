// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/record"
)

// The allocation ceilings of the functions of Calls.
const (
	// addAllocs are the allocations of Add on a recorder's Calls, which
	// encode the record of a passing call with a call site.
	addAllocs = 9
	// linesAllocs are the allocations of Lines: the slice it returns.
	linesAllocs = 1
)

// TestCalls checks how Calls find, number, keep and cut the records of
// calls.
func TestCalls(t *testing.T) {
	t.Parallel()

	t.Run("Of", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the Calls of a seat that keeps the records of its calls", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			assert.True(t, record.Of(k) == &k.calls, "the seat's Calls")
		})
		t.Run("returns nil for a seat whose Calls record nothing", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, record.Of(&keeping{}), "no Calls")
		})
		t.Run("returns nil for the seat of a run whose call is not recorded", func(t *testing.T) {
			t.Parallel()
			b := &body{}
			record.Run(&b.calls, nil, nil)
			assert.Nil(t, record.Of(b), "no Calls")
		})
		t.Run("returns nil for a seat without Calls or attributes", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, record.Of(struct{}{}), "no Calls")
		})
	})

	t.Run("Add", func(t *testing.T) {
		t.Parallel()

		t.Run("numbers the calls of a recorder from 1 in the order of their verdicts", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			record.Add(&k.calls, call("true", "first"))
			record.Add(&k.calls, call("false", "second"))
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 2, "two records")
			assert.Equal(t, decoded(t, lines[0])["seq"], any(1.0), "the first number")
			assert.Equal(t, decoded(t, lines[1])["seq"], any(2.0), "the second number")
			assert.Equal(t, decoded(t, lines[1])["contract"], any("second"), "the second call")
		})
		t.Run("keeps nothing for Calls that record nothing", func(t *testing.T) {
			t.Parallel()
			k := &keeping{}
			record.Add(&k.calls, call("true", "dropped"))
			assert.Empty(t, record.Lines(&k.calls), "no record")
		})
	})

	t.Run("Keep", func(t *testing.T) {
		t.Parallel()

		t.Run("drops the records that the Calls kept before", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			record.Add(&k.calls, call("true", "before"))
			record.Keep(&k.calls)
			record.Add(&k.calls, call("true", "after"))
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 1, "one record")
			assert.Equal(t, decoded(t, lines[0])["seq"], any(1.0), "numbered from 1 again")
		})
	})

	t.Run("Lines", func(t *testing.T) {
		t.Parallel()

		t.Run("leaves out a call that is still running its body", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			record.Begin(k)
			record.Add(&k.calls, call("true", "after the slot"))
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 1, "one record")
			assert.Equal(t, decoded(t, lines[0])["seq"], any(2.0), "the number after the slot's")
		})
		t.Run("returns a copy that later calls do not change", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			record.Add(&k.calls, call("true", "kept"))
			lines := record.Lines(&k.calls)
			record.Keep(&k.calls)
			assert.Length(t, lines, 1, "the copy keeps its record")
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps the calls of a run unnumbered until their call takes them", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			slot := record.Begin(k)
			b := &body{}
			record.Run(&b.calls, slot, nil)
			record.Add(&b.calls, call("true", "in the body"))
			assert.Empty(t, record.Lines(&k.calls), "no record before the take")
			slot.Take(&b.calls, record.NoPhase)
			assert.Length(t, record.Lines(&k.calls), 1, "the body's record")
		})
		t.Run("drops what the Calls kept before", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			record.Add(&k.calls, call("true", "kept"))
			record.Run(&k.calls, nil, nil)
			assert.Empty(t, record.Lines(&k.calls), "no record")
			assert.Nil(t, record.Of(k), "the Calls record nothing")
		})
	})

	t.Run("Cut", func(t *testing.T) {
		t.Parallel()

		t.Run("drops the calls made at the step and after it", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			slot := record.Begin(k)
			steps := 0
			b := &body{}
			record.Run(&b.calls, slot, func() int { return steps })
			for i, contract := range []string{"at step 0", "at step 1", "at step 2", "at step 3"} {
				steps = i
				record.Add(&b.calls, call("true", contract))
			}
			record.Cut(&b.calls, 2)
			slot.Take(&b.calls, record.Random)
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 2, "the calls before step 2")
			assert.Equal(t, decoded(t, lines[1])["contract"], any("at step 1"), "the last call kept")
		})
	})
}

// TestCallsAllocs checks the ceilings of the functions of Calls.
func TestCallsAllocs(t *testing.T) {
	k := &keeping{}
	c := record.Call{
		Assertion: "equal",
		Contract:  "c",
		Verdict:   record.Pass,
		Where:     record.Where{File: "a_test.go", Line: 1},
	}
	record.Keep(&k.calls)
	record.Add(&k.calls, c)
	b := &body{}
	assert.MaxAllocs(t, func() { _ = record.Of(k) }, 0, "Of allocates nothing")
	assert.MaxAllocs(t, func() {
		record.Keep(&k.calls)
		record.Add(&k.calls, c)
	}, addAllocs, "Add allocates its entry and the encoded record")
	assert.MaxAllocs(t, func() { record.Keep(&k.calls) }, 0, "Keep allocates nothing")
	assert.MaxAllocs(t, func() { _ = record.Lines(&k.calls) }, linesAllocs, "Lines allocates its slice")
	assert.MaxAllocs(t, func() { record.Run(&b.calls, nil, nil) }, 0, "Run allocates nothing")
	assert.MaxAllocs(t, func() { record.Cut(&b.calls, 0) }, 0, "Cut allocates nothing")
}

// BenchmarkCalls measures the functions of Calls on a recorder's Calls and
// on the Calls of a run.
func BenchmarkCalls(b *testing.B) {
	k := &keeping{}
	c := record.Call{
		Assertion: "equal",
		Contract:  "c",
		Verdict:   record.Pass,
		Where:     record.Where{File: "a_test.go", Line: 1},
	}
	record.Keep(&k.calls)
	record.Add(&k.calls, c)
	run := &body{}

	b.Run("Of", func(b *testing.B) {
		bc := bench.Start(b).MaxAllocs(0)
		defer bc.End()
		for bc.Loop() {
			_ = record.Of(k)
		}
	})
	b.Run("Add", func(b *testing.B) {
		bc := bench.Start(b).MaxAllocs(addAllocs)
		defer bc.End()
		for bc.Loop() {
			record.Keep(&k.calls)
			record.Add(&k.calls, c)
		}
	})
	b.Run("Keep", func(b *testing.B) {
		bc := bench.Start(b).MaxAllocs(0)
		defer bc.End()
		for bc.Loop() {
			record.Keep(&k.calls)
		}
	})
	b.Run("Lines", func(b *testing.B) {
		record.Add(&k.calls, c)
		bc := bench.Start(b).MaxAllocs(linesAllocs)
		defer bc.End()
		for bc.Loop() {
			_ = record.Lines(&k.calls)
		}
	})
	b.Run("Run", func(b *testing.B) {
		bc := bench.Start(b).MaxAllocs(0)
		defer bc.End()
		for bc.Loop() {
			record.Run(&run.calls, nil, nil)
		}
	})
	b.Run("Cut", func(b *testing.B) {
		bc := bench.Start(b).MaxAllocs(0)
		defer bc.End()
		for bc.Loop() {
			record.Cut(&run.calls, 0)
		}
	})
}
