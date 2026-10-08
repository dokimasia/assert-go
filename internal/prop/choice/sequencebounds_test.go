// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestSequenceBounds checks the construction, the element bounds, the
// target, the admission and the replay of sequence bounds.
func TestSequenceBounds(t *testing.T) {
	t.Parallel()

	t.Run("NewSequenceBounds", func(t *testing.T) {
		t.Parallel()

		t.Run("returns ErrEmpty for k of 0", func(t *testing.T) {
			t.Parallel()
			_, err := choice.NewSequenceBounds(0, sizes(t, 0, 1))
			assert.ErrorIs(t, err, choice.ErrEmpty, "the refusal")
		})

		t.Run("returns bounds that report what they were given", func(t *testing.T) {
			t.Parallel()
			s := sizes(t, 3, 8)
			b, err := choice.NewSequenceBounds(byteK, s)
			assert.NoError(t, err, "the bounds are valid")
			assert.Equal(t, b.K(), uint32(byteK), "k")
			assert.Equal(t, b.Sizes(), s, "the sizes")
		})
	})

	t.Run("MustSequenceBounds", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the bounds that NewSequenceBounds returns", func(t *testing.T) {
			t.Parallel()
			got := choice.MustSequenceBounds(byteK, sizes(t, 0, 8))
			assert.Equal(t, got, sequenceBounds(t, byteK, 0, 8), "the bounds")
		})

		t.Run("panics with ErrEmpty for k of 0", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { choice.MustSequenceBounds(0, sizes(t, 0, 1)) }, "invalid bounds")
			err, ok := got.(error)
			assert.True(t, ok, "the panic value is the error")
			assert.ErrorIs(t, err, choice.ErrEmpty, "the refusal")
		})
	})

	t.Run("Element", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the bounds from 0 to k - 1", func(t *testing.T) {
			t.Parallel()
			e := sequenceBounds(t, byteK, 0, 8).Element()
			assert.Equal(t, [2]choice.Int{e.Lo(), e.Hi()}, [2]choice.Int{{}, choice.UintOf(byteK - 1)}, "one byte")
		})
	})

	t.Run("Target", func(t *testing.T) {
		t.Parallel()

		t.Run("returns as many zeros as the minimum length", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, sequenceBounds(t, byteK, 3, 8).Target(), []uint32{0, 0, 0}, "the simplest sequence")
		})

		t.Run("returns nil for a minimum of zero", func(t *testing.T) {
			t.Parallel()
			assert.True(t, sequenceBounds(t, byteK, 0, 8).Target() == nil, "the empty sequence allocates nothing")
		})
	})

	t.Run("Admits", func(t *testing.T) {
		t.Parallel()

		b := sequenceBounds(t, 3, 1, 2)
		tests := []struct {
			name string
			give []uint32
			want bool
		}{
			{name: "reports true for a sequence of the minimum length", give: []uint32{2}, want: true},
			{name: "reports true for a sequence of the maximum length", give: []uint32{2, 1}, want: true},
			{name: "reports false below the minimum", give: nil, want: false},
			{name: "reports false above the maximum", give: []uint32{0, 0, 0}, want: false},
			{name: "reports false for an element of k", give: []uint32{3}, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, b.Admits(tt.give), tt.want, "whether the sequence is in the bounds")
			})
		}
	})

	t.Run("Coerce", func(t *testing.T) {
		t.Parallel()

		b := sequenceBounds(t, 10, 2, 4)
		tests := []struct {
			name string
			give choice.Choice
			want []uint32
		}{
			{
				name: "cuts a longer sequence to the maximum",
				give: sequenceChoice(1, 2, 3, 4, 5),
				want: []uint32{1, 2, 3, 4},
			},
			{name: "extends a shorter sequence with zeros", give: sequenceChoice(7), want: []uint32{7, 0}},
			{name: "replaces an element of k or more with 0", give: sequenceChoice(11, 10, 3), want: []uint32{0, 0, 3}},
			{
				name: "returns the target for an integer",
				give: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(5)},
				want: []uint32{0, 0},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, b.Coerce(tt.give), tt.want, "the fitted sequence")
			})
		}

		t.Run("returns the recorded slice when it fits", func(t *testing.T) {
			t.Parallel()
			recorded := sequenceChoice(1, 2, 3)
			got := b.Coerce(recorded)
			assert.True(t, &got[0] == &recorded.Sequence[0], "a fitting sequence is shared, not copied")
		})

		t.Run("cuts nothing from an unbounded sequence", func(t *testing.T) {
			t.Parallel()
			unbounded, err := choice.NewSequenceBounds(2, unboundedSizes(t, 0))
			assert.NoError(t, err, "the bounds are valid")
			assert.Equal(
				t,
				unbounded.Coerce(sequenceChoice(1, 5, 1)),
				[]uint32{1, 0, 1},
				"only the element of k changes",
			)
		})
	})
}

// TestSequenceBoundsAllocs checks that the constructor and every method
// of SequenceBounds that returns no new sequence allocate nothing.
func TestSequenceBoundsAllocs(t *testing.T) {
	s := sizes(t, 0, 8)
	b := sequenceBounds(t, byteK, 0, 8)
	recorded := sequenceChoice(1, 2, 3)
	assert.MaxAllocs(t, func() { _, _ = choice.NewSequenceBounds(byteK, s) }, 0, "NewSequenceBounds allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.MustSequenceBounds(byteK, s) }, 0, "MustSequenceBounds allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.K() }, 0, "K allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Sizes() }, 0, "Sizes allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Element() }, 0, "Element allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Target() }, 0, "Target of a minimum of zero allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Admits(recorded.Sequence) }, 0, "Admits allocates nothing")
	assert.MaxAllocs(t, func() { _ = b.Coerce(recorded) }, 0, "Coerce of a fitting sequence allocates nothing")
}

// BenchmarkSequenceBounds measures the constructor and each method of
// SequenceBounds. A target of a minimum above zero and the coercion of a
// sequence that does not fit allocate one slice.
func BenchmarkSequenceBounds(b *testing.B) {
	s := sizes(b, 3, 8)
	bounds := sequenceBounds(b, byteK, 3, 8)

	b.Run("NewSequenceBounds", func(b *testing.B) {
		var got choice.SequenceBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got, _ = choice.NewSequenceBounds(byteK, s)
		}
		assert.Equal(b, got, bounds, "the bounds")
	})

	b.Run("MustSequenceBounds", func(b *testing.B) {
		var got choice.SequenceBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.MustSequenceBounds(byteK, s)
		}
		assert.Equal(b, got, bounds, "the bounds")
	})

	b.Run("K", func(b *testing.B) {
		var got uint32
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.K()
		}
		assert.Equal(b, got, uint32(byteK), "k")
	})

	b.Run("Sizes", func(b *testing.B) {
		var got choice.Sizes
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Sizes()
		}
		assert.Equal(b, got, s, "the sizes")
	})

	b.Run("Element", func(b *testing.B) {
		var got choice.IntegerBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Element()
		}
		assert.Equal(b, got.Hi(), choice.UintOf(byteK-1), "the largest byte")
	})

	b.Run("Target", func(b *testing.B) {
		var got []uint32
		c := bench.Start(b).MaxAllocs(2)
		defer c.End()
		for c.Loop() {
			got = bounds.Target()
		}
		assert.Equal(b, got, []uint32{0, 0, 0}, "three zeros")
	})

	b.Run("Admits", func(b *testing.B) {
		var got bool
		sequence := []uint32{1, 2, 3}
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Admits(sequence)
		}
		assert.True(b, got, "the sequence is in the bounds")
	})

	b.Run("Coerce", func(b *testing.B) {
		var got []uint32
		recorded := sequenceChoice(1, 2)
		c := bench.Start(b).MaxAllocs(2)
		defer c.End()
		for c.Loop() {
			got = bounds.Coerce(recorded)
		}
		assert.Equal(b, got, []uint32{1, 2, 0}, "the sequence extended to the minimum")
	})
}
