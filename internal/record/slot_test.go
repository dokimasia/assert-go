// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/record"
)

// The allocation ceilings of a slot.
const (
	// beginAllocs are the allocations of Begin on a recorder: the slot and
	// its entry.
	beginAllocs = 2
	// writeAllocs are the allocations of Write on a recorder: the encoded
	// record.
	writeAllocs = 6
)

// TestSlot checks how a call that runs a body takes its number, the calls
// of its runs and its record.
func TestSlot(t *testing.T) {
	t.Parallel()

	t.Run("Begin", func(t *testing.T) {
		t.Parallel()

		t.Run("returns nil for a seat whose calls are not recorded", func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, record.Begin(struct{}{}), "no slot")
		})
		t.Run("numbers the call before the calls of its body", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			slot := record.Begin(k)
			b := &body{}
			record.Run(&b.calls, slot, nil)
			record.Add(&b.calls, call("true", "in the body"))
			slot.Take(&b.calls, record.NoPhase)
			slot.Write(call("eventually", "it converges"))
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 2, "two records")
			first, second := decoded(t, lines[0]), decoded(t, lines[1])
			assert.Equal(t, first["assertion"], any("eventually"), "the call that ran the body is first")
			assert.Equal(t, first["seq"], any(1.0), "its number")
			assert.Equal(t, second["seq"], any(2.0), "the body's call")
			assert.Equal(t, second["parent"], any(1.0), "under the call that ran the body")
			assert.Equal(t, second["run"], any(1.0), "in the first run")
			_, phased := second["phase"]
			assert.False(t, phased, "a call outside a property's case states no phase")
		})
	})

	t.Run("Take", func(t *testing.T) {
		t.Parallel()

		t.Run("numbers every run, one without calls included", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			slot := record.Begin(k)
			b := &body{}
			for i := range 3 {
				record.Run(&b.calls, slot, nil)
				if i != 1 {
					record.Add(&b.calls, call("true", "a case"))
				}
				slot.Take(&b.calls, record.Random)
			}
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 2, "the calls of runs 1 and 3")
			assert.Equal(t, decoded(t, lines[0])["run"], any(1.0), "the first run")
			assert.Equal(t, decoded(t, lines[1])["run"], any(3.0), "the third run")
			assert.Equal(t, decoded(t, lines[1])["phase"], any("random"), "the phase of the run")
		})
		t.Run("keeps the parent of a call in a body inside the run", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			property := record.Begin(k)
			c := &body{}
			record.Run(&c.calls, property, nil)
			eventually := record.Begin(c)
			attempt := &body{}
			record.Run(&attempt.calls, eventually, nil)
			record.Add(&attempt.calls, call("equal", "in an attempt"))
			eventually.Take(&attempt.calls, record.NoPhase)
			eventually.Write(call("eventually", "it converges"))
			property.Take(&c.calls, record.Simplest)
			property.Write(call("prop-for-all", "it holds"))
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 3, "three records")
			inner, deepest := decoded(t, lines[1]), decoded(t, lines[2])
			assert.Equal(t, inner["assertion"], any("eventually"), "the call in the case")
			assert.Equal(t, inner["parent"], any(1.0), "under the property")
			assert.Equal(t, inner["phase"], any("simplest"), "in the property's case")
			assert.Equal(t, deepest["parent"], any(2.0), "under the call in the case")
			_, phased := deepest["phase"]
			assert.False(t, phased, "a call in an attempt states no phase")
		})
		t.Run("does nothing for a nil slot", func(t *testing.T) {
			t.Parallel()
			var slot *record.Slot
			b := &body{}
			slot.Take(&b.calls, record.Random)
			slot.Write(call("true", "nothing"))
			assert.Nil(t, record.Of(b), "nothing recorded")
		})
	})

	t.Run("Write", func(t *testing.T) {
		t.Parallel()

		t.Run("writes the record of a slot in a run when its run is taken", func(t *testing.T) {
			t.Parallel()
			k := newKeeping()
			property := record.Begin(k)
			c := &body{}
			record.Run(&c.calls, property, nil)
			record.Begin(c).Write(call("rejects", "the check fails"))
			property.Take(&c.calls, record.Stored)
			lines := record.Lines(&k.calls)
			assert.Length(t, lines, 1, "the record of the slot in the run")
			assert.Equal(t, decoded(t, lines[0])["assertion"], any("rejects"), "the slot's call")
		})
	})
}

// TestSlotAllocs checks the ceilings of a slot on a recorder.
func TestSlotAllocs(t *testing.T) {
	k := newKeeping()
	c := call("eventually", "it converges")
	b := &body{}
	assert.MaxAllocs(t, func() { _ = record.Begin(k) }, beginAllocs, "Begin allocates the slot and its entry")
	slot := record.Begin(k)
	slot.Write(c)
	assert.MaxAllocs(t, func() { slot.Write(c) }, writeAllocs, "Write allocates the encoded record")
	assert.MaxAllocs(t, func() { slot.Take(&b.calls, record.NoPhase) }, 0, "Take of an empty run allocates nothing")
}

// BenchmarkSlot measures Begin, Take and Write on a recorder.
func BenchmarkSlot(b *testing.B) {
	k := newKeeping()
	c := call("eventually", "it converges")
	run := &body{}

	b.Run("Begin", func(b *testing.B) {
		bc := bench.Start(b).MaxAllocs(beginAllocs)
		defer bc.End()
		for bc.Loop() {
			_ = record.Begin(k)
		}
	})
	b.Run("Take", func(b *testing.B) {
		slot := record.Begin(k)
		bc := bench.Start(b).MaxAllocs(0)
		defer bc.End()
		for bc.Loop() {
			slot.Take(&run.calls, record.NoPhase)
		}
	})
	b.Run("Write", func(b *testing.B) {
		slot := record.Begin(k)
		slot.Write(c)
		bc := bench.Start(b).MaxAllocs(writeAllocs)
		defer bc.End()
		for bc.Loop() {
			slot.Write(c)
		}
	})
}
