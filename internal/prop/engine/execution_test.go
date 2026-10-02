// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// invalidStatus is the first value past the five statuses.
const invalidStatus engine.Status = 5

// The allocations of one execution of a body that draws one integer,
// measured.
const (
	// generateAllocs are the allocations of a generated case, its provider
	// and its record of reused values included.
	generateAllocs = 12
	// bridgeAllocs are the allocations of a case decoded from bytes.
	bridgeAllocs = 9
)

// TestExecution checks how one call of a body ends, under each of the
// three ways a case gets its values, and pins each status's spelling.
func TestExecution(t *testing.T) {
	t.Parallel()

	whole := engine.Integer[uint64](0, math.MaxUint64)

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give engine.Status
			want bool
		}{
			{name: "reports true for CasePassed", give: engine.CasePassed, want: true},
			{name: "reports true for CaseDiverged", give: engine.CaseDiverged, want: true},
			{name: "reports false past CaseDiverged", give: invalidStatus, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.Valid(), tt.want, "whether the value is a status")
			})
		}
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give engine.Status
			want string
		}{
			{name: "returns passed for CasePassed", give: engine.CasePassed, want: "passed"},
			{name: "returns failed for CaseFailed", give: engine.CaseFailed, want: "failed"},
			{name: "returns rejected for CaseRejected", give: engine.CaseRejected, want: "rejected"},
			{name: "returns repeated for CaseRepeated", give: engine.CaseRepeated, want: "repeated"},
			{name: "returns diverged for CaseDiverged", give: engine.CaseDiverged, want: "diverged"},
			{name: "returns Status(5) for a value that is no status", give: invalidStatus, want: "Status(5)"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the status's spelling")
			})
		}
	})

	t.Run("Generate", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the pinned values of the first cases of seed 7", func(t *testing.T) {
			t.Parallel()
			values, _ := generated(whole, 7, 6)
			assert.Equal(t, values, []uint64{17930806776211019302, 35, 37324, 15, 9, 38313},
				"the values of the first six cases")
		})

		t.Run("returns a passed execution of a body that does not fail", func(t *testing.T) {
			t.Parallel()
			e := engine.Generate(func(c *engine.Case) { engine.Draw(c, whole, drawn) }, 7, 2, nil)
			assert.Equal(t, e.Status, engine.CasePassed, "the case passes")
			assert.Equal(t, drawValues(e.Case.Draws()), []any{uint64(37324)}, "case 2 of seed 7")
		})

		t.Run("returns the clock it is given to the case", func(t *testing.T) {
			t.Parallel()
			start := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
			var got time.Time
			engine.Generate(func(c *engine.Case) { got = c.Clock().Now() }, 7, 0, assert.NewControlled(start))
			assert.Equal(t, got, start, "the controlled clock's time")
		})
	})

	t.Run("Replay", func(t *testing.T) {
		t.Parallel()

		t.Run("returns CaseRejected for a case past MaxChoices", func(t *testing.T) {
			t.Parallel()
			var after bool
			e := engine.Replay(func(c *engine.Case) {
				source := c.Rand()
				for range engine.MaxChoices + 1 {
					source.Uint64()
				}
				after = true
			}, nil, nil)
			assert.Equal(t, e.Status, engine.CaseRejected, "the case is rejected")
			assert.False(t, after, "the body ends at the choice past the cap")
			assert.Length(t, e.Case.Choices(), engine.MaxChoices, "the choices up to the cap")
		})

		tests := []struct {
			name     string
			elements int
			want     engine.Status
		}{
			{
				name:     "returns CaseRejected for a sequence whose elements take the case past MaxChoices",
				elements: engine.MaxChoices,
				want:     engine.CaseRejected,
			},
			{
				name:     "returns CasePassed for a sequence whose elements fill MaxChoices",
				elements: engine.MaxChoices - 1,
				want:     engine.CasePassed,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g := engine.Bytes(sizes(t, tt.elements, tt.elements))
				e := engine.Replay(func(c *engine.Case) { engine.Draw(c, g, drawn) }, nil, nil)
				assert.Equal(t, e.Status, tt.want, "one choice for the sequence and one for each element")
			})
		}

		t.Run("returns CaseFailed for a body that fails and then rejects", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, false)
				c.Assume(false)
			}, nil, nil)
			assert.Equal(t, e.Status, engine.CaseFailed, "the failure outweighs the rejection")
			assert.Equal(t, e.Identity.Assertion, "equal", "the record's assertion")
		})

		t.Run("returns the identity of a record over a later panic", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, false)
				panic("late")
			}, nil, nil)
			assert.Equal(t, e.Identity, engine.Identity{Assertion: "equal", Where: reported.Where}, "the first record")
		})

		t.Run("returns no panic value for a case that a record failed first", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(c *engine.Case) {
				c.Report(reported, false)
				panic("late")
			}, nil, nil)
			assert.Nil(t, e.Panic, "no panic value")
			assert.Nil(t, e.Stack, "no stack")
		})

		t.Run("returns the value and the stack of a panic", func(t *testing.T) {
			t.Parallel()
			e := engine.Replay(func(*engine.Case) { panic("boom") }, nil, nil)
			assert.Equal(t, e.Status, engine.CaseFailed, "the case fails")
			assert.Equal[any](t, e.Panic, "boom", "the panic's value")
			assert.Contains(t, string(e.Stack), "TestExecution", "the stack holds the body's frame")
		})
	})

	t.Run("Bridge", func(t *testing.T) {
		t.Parallel()

		t.Run("returns a passed execution of the case its bytes decode to", func(t *testing.T) {
			t.Parallel()
			g := engine.Integer(0, 1000)
			e := engine.Bridge(func(c *engine.Case) { engine.Draw(c, g, drawn) }, []byte{0xe8, 0x03}, nil)
			assert.Equal(t, e.Status, engine.CasePassed, "the case passes")
			assert.Equal(t, drawValues(e.Case.Draws()), []any{1000}, "two bytes, little-endian")
		})
	})
}

// TestExecutionZeroAlloc checks the allocation ceilings of the three ways
// to execute a body, and that no method of Status allocates.
func TestExecutionZeroAlloc(t *testing.T) {
	digit := engine.Integer(0, 9)
	body := func(c *engine.Case) { engine.Draw(c, digit, drawn) }
	seven, bytes := integers(7), []byte{7}
	assert.MaxAllocs(t, func() { engine.Generate(body, 7, 0, nil) }, generateAllocs, "a generated case")
	assert.MaxAllocs(t, func() { engine.Replay(body, seven, nil) }, drawAllocs, "a replayed case")
	assert.MaxAllocs(t, func() { engine.Bridge(body, bytes, nil) }, bridgeAllocs, "a case decoded from bytes")
	assert.MaxAllocs(t, func() { _ = engine.CaseDiverged.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _ = engine.CaseDiverged.String() }, 0, "String allocates nothing")
}

// BenchmarkExecution measures one execution of a body that draws one
// integer under each of the three ways a case gets its values, and each
// method of Status.
func BenchmarkExecution(b *testing.B) {
	digit := engine.Integer(0, 9)
	body := func(c *engine.Case) { engine.Draw(c, digit, drawn) }

	b.Run("Generate", func(b *testing.B) {
		var got engine.Execution
		c := bench.Start(b).MaxAllocs(generateAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Generate(body, 7, 0, nil)
		}
		assert.Equal(b, got.Status, engine.CasePassed, "the case passes")
	})

	b.Run("Replay", func(b *testing.B) {
		var got engine.Execution
		seven := integers(7)
		c := bench.Start(b).MaxAllocs(drawAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(body, seven, nil)
		}
		assert.Equal(b, got.Status, engine.CasePassed, "the case passes")
	})

	b.Run("Bridge", func(b *testing.B) {
		var got engine.Execution
		bytes := []byte{7}
		c := bench.Start(b).MaxAllocs(bridgeAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Bridge(body, bytes, nil)
		}
		assert.Equal(b, got.Status, engine.CasePassed, "the case passes")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.CaseDiverged.Valid()
		}
		assert.True(b, got, "CaseDiverged is a status")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = engine.CaseDiverged.String()
		}
		assert.Equal(b, got, "diverged", "the status's spelling")
	})
}
