// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/record"
)

// TestObserve checks which calls an observer of a seat's chain receives, and
// when.
func TestObserve(t *testing.T) {
	t.Parallel()

	t.Run("Observer", func(t *testing.T) {
		t.Parallel()

		t.Run("receives the assertion and the contract of a call", func(t *testing.T) {
			t.Parallel()
			w := &watching{}
			record.Add(record.Of(w), call("equal", "the totals match"))
			assert.Equal(t, w.seen, []record.Observation{{Assertion: "equal", Contract: "the totals match"}},
				"the call")
		})
		t.Run("receives a call that runs a body when the call writes its record", func(t *testing.T) {
			t.Parallel()
			w := &watching{}
			slot := record.Begin(w)
			assert.Empty(t, w.observed(), "nothing before the record")
			slot.Write(call("eventually", "it converges"))
			assert.Equal(t, w.observed(), []string{"it converges"}, "the call")
		})
		t.Run("receives the calls of a body when the call takes them", func(t *testing.T) {
			t.Parallel()
			w := &watching{}
			slot := record.Begin(w)
			b := &body{}
			record.Run(&b.calls, slot, nil)
			record.Add(&b.calls, call("true", "in the body"))
			assert.Empty(t, w.observed(), "nothing before the take")
			slot.Take(&b.calls, record.Random)
			slot.Write(call("prop-for-all", "it holds"))
			assert.Equal(t, w.observed(), []string{"in the body", "it holds"}, "the body's call, then the property")
		})
		t.Run("receives the calls of a body once when a recorder keeps them", func(t *testing.T) {
			t.Parallel()
			r := assert.NewRecorder()
			w := &watching{TB: r}
			slot := record.Begin(w)
			b := &body{}
			record.Run(&b.calls, slot, nil)
			record.Add(&b.calls, call("true", "in the body"))
			slot.Take(&b.calls, record.Random)
			slot.Write(call("prop-for-all", "it holds"))
			assert.Equal(t, w.observed(), []string{"in the body", "it holds"}, "each call once")
			assert.Length(t, r.Records(), 2, "the recorder keeps both records")
		})
		t.Run("receives the calls of a body inside a run of a body", func(t *testing.T) {
			t.Parallel()
			w := &watching{}
			property := record.Begin(w)
			c := &body{}
			record.Run(&c.calls, property, nil)
			eventually := record.Begin(c)
			attempt := &body{}
			record.Run(&attempt.calls, eventually, nil)
			record.Add(&attempt.calls, call("equal", "in an attempt"))
			eventually.Take(&attempt.calls, record.NoPhase)
			eventually.Write(call("eventually", "it converges"))
			assert.Empty(t, w.observed(), "nothing before the case is taken")
			property.Take(&c.calls, record.Simplest)
			property.Write(call("prop-for-all", "it holds"))
			assert.Equal(t, w.observed(), []string{"it converges", "in an attempt", "it holds"},
				"the calls of the case in the order of their numbers, then the property")
		})
	})
}
