// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice_test

import (
	"encoding/json"
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/choice"
)

// TestBounds checks that Bounds dispatch every operation to the bounds of
// their kind, and that they compare as values.
func TestBounds(t *testing.T) {
	t.Parallel()

	integer := choice.OfInteger(signedBounds(t, 3, 9))
	float := choice.OfFloat(floatBounds(t, 1, 2, choice.ExcludeNaN, choice.Width64))
	sequence := choice.OfSequence(sequenceBounds(t, byteK, 1, 4))

	t.Run("Kind", func(t *testing.T) {
		t.Parallel()
		got := []choice.Kind{integer.Kind(), float.Kind(), sequence.Kind()}
		assert.Equal(t, got, []choice.Kind{choice.Integer, choice.Float, choice.Sequence}, "the kind of each")
	})

	t.Run("Integer", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, integer.Integer(), signedBounds(t, 3, 9), "the integer bounds")
	})

	t.Run("Float", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, float.Float().Hi(), 2.0, "the float bounds")
	})

	t.Run("Sequence", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, sequence.Sequence().Sizes().Min(), 1, "the sequence bounds")
	})

	t.Run("Target", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			b    choice.Bounds
			want choice.Choice
		}{
			{
				name: "returns the integer target",
				b:    integer,
				want: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(3)},
			},
			{name: "returns the float target", b: float, want: choice.Choice{Kind: choice.Float, Float: 1}},
			{name: "returns the sequence target", b: sequence, want: sequenceChoice(0)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.True(t, tt.b.Target().Equal(tt.want), "the simplest choice")
			})
		}
	})

	t.Run("Admits", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			b    choice.Bounds
			give choice.Choice
			want bool
		}{
			{
				name: "reports true for an integer inside integer bounds",
				b:    integer,
				give: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(9)},
				want: true,
			},
			{
				name: "reports false for an integer outside integer bounds",
				b:    integer,
				give: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(10)},
				want: false,
			},
			{
				name: "reports true for a float inside float bounds",
				b:    float,
				give: choice.Choice{Kind: choice.Float, Float: 1.5},
				want: true,
			},
			{
				name: "reports false for a float outside float bounds",
				b:    float,
				give: choice.Choice{Kind: choice.Float, Float: 3},
				want: false,
			},
			{
				name: "reports false for a sequence outside sequence bounds",
				b:    sequence,
				give: sequenceChoice(),
				want: false,
			},
			{
				name: "reports true for a sequence inside sequence bounds",
				b:    sequence,
				give: sequenceChoice(255),
				want: true,
			},
			{
				name: "reports false for a choice of another kind",
				b:    sequence,
				give: choice.Choice{Kind: choice.Float, Float: 1.5},
				want: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.b.Admits(tt.give), tt.want, "whether the bounds admit the choice")
			})
		}
	})

	t.Run("Coerce", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			b    choice.Bounds
			give choice.Choice
			want choice.Choice
		}{
			{
				name: "returns the integer target for a float",
				b:    integer,
				give: choice.Choice{Kind: choice.Float, Float: 4},
				want: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(3)},
			},
			{
				name: "returns a recorded float inside float bounds",
				b:    float,
				give: choice.Choice{Kind: choice.Float, Float: 1.5},
				want: choice.Choice{Kind: choice.Float, Float: 1.5},
			},
			{
				name: "returns a sequence fitted to sequence bounds",
				b:    sequence,
				give: sequenceChoice(1, 2, 3, 4, 5),
				want: sequenceChoice(1, 2, 3, 4),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.True(t, tt.b.Coerce(tt.give).Equal(tt.want), "the replayed choice")
			})
		}
	})

	t.Run("Key", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			b    choice.Bounds
			give choice.Choice
			want choice.Key
		}{
			{
				name: "returns the integer key under integer bounds",
				b:    integer,
				give: choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(5)},
				want: signedBounds(t, 3, 9).Key(choice.IntOf(5)),
			},
			{
				name: "returns the float key under float bounds",
				b:    float,
				give: choice.Choice{Kind: choice.Float, Float: 1.5},
				want: choice.FloatKey(1.5),
			},
			{
				name: "returns the sequence key under sequence bounds",
				b:    sequence,
				give: sequenceChoice(1, 2),
				want: choice.SequenceKey([]uint32{1, 2}),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.b.Key(tt.give).Compare(tt.want), 0, "the key of the bounds' kind")
			})
		}
	})

	t.Run("OfInteger", func(t *testing.T) {
		t.Parallel()

		t.Run("returns bounds equal under == to bounds of the same integer bounds", func(t *testing.T) {
			t.Parallel()
			assert.True(t, choice.OfInteger(signedBounds(t, 3, 9)) == integer, "the same integer bounds")
		})
	})

	t.Run("OfFloat", func(t *testing.T) {
		t.Parallel()

		t.Run("returns bounds unequal under == to integer bounds of zero values", func(t *testing.T) {
			t.Parallel()
			zero := choice.OfInteger(choice.IntegerBounds{})
			assert.False(t, zero == choice.OfFloat(choice.FloatBounds{}), "integer bounds are not float bounds")
		})
	})

	t.Run("OfSequence", func(t *testing.T) {
		t.Parallel()

		t.Run("returns bounds whose kind is Sequence", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, choice.OfSequence(choice.SequenceBounds{}).Kind(), choice.Sequence, "sequence bounds")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Bounds
			want string
		}{
			{name: "returns the range of integer bounds", give: integer, want: "integer in [3, 9]"},
			{
				name: "returns the range and the width of float bounds",
				give: float,
				want: "float in [1, 2] of width 64",
			},
			{
				name: "returns the range, the width and NaN of float bounds that admit NaN",
				give: choice.OfFloat(floatBounds(t, -0.5, math.Inf(1), choice.AdmitNaN, choice.Width32)),
				want: "float in [-0.5, +Inf] of width 32 or NaN",
			},
			{
				name: "returns the sizes and the elements of bounded sequence bounds",
				give: sequence,
				want: "sequence of 1 to 4 values below 256",
			},
			{
				name: "returns the minimum size and the elements of unbounded sequence bounds",
				give: choice.OfSequence(choice.MustSequenceBounds(byteK, unboundedSizes(t, 2))),
				want: "sequence of 2 or more values below 256",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the text of the request")
			})
		}
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give choice.Bounds
			want string
		}{
			{name: "returns the range of integer bounds", give: integer, want: `{"kind":"integer","min":3,"max":9}`},
			{
				name: "returns a decimal string for an integer bound beyond 2^53 - 1",
				give: choice.OfInteger(signedBounds(t, math.MinInt64, math.MaxInt64)),
				want: `{"kind":"integer","min":"-9223372036854775808","max":"9223372036854775807"}`,
			},
			{
				name: "returns a decimal string for an unsigned bound beyond the int64 range",
				give: choice.OfInteger(choice.MustIntegerBounds(choice.Int{}, choice.UintOf(math.MaxUint64))),
				want: `{"kind":"integer","min":0,"max":"18446744073709551615"}`,
			},
			{
				name: "returns the range, NaN and the width of float bounds",
				give: float,
				want: `{"kind":"float","min":1,"max":2,"allow_nan":false,"width":64}`,
			},
			{
				name: "returns the name of an infinite float bound",
				give: choice.OfFloat(floatBounds(t, -0.5, math.Inf(1), choice.AdmitNaN, choice.Width32)),
				want: `{"kind":"float","min":-0.5,"max":"Inf","allow_nan":true,"width":32}`,
			},
			{
				name: "returns the elements and the sizes of bounded sequence bounds",
				give: sequence,
				want: `{"kind":"sequence","k":256,"min_size":1,"max_size":4}`,
			},
			{
				name: "returns null for the greatest size of unbounded sequence bounds",
				give: choice.OfSequence(choice.MustSequenceBounds(byteK, unboundedSizes(t, 2))),
				want: `{"kind":"sequence","k":256,"min_size":2,"max_size":null}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := json.Marshal(tt.give)
				assert.NoError(t, err, "the bounds are JSON")
				assert.Equal(t, string(got), tt.want, "the corpus form of the request")
			})
		}
	})
}

// marshalAllocs are the allocations of MarshalJSON on integer bounds of
// the whole signed range.
const marshalAllocs = 15

// TestBoundsAllocs checks that no operation of Bounds allocates, but
// for a sequence's target, the coercion of a sequence that does not fit,
// the text that String returns, and the JSON that MarshalJSON returns.
func TestBoundsAllocs(t *testing.T) {
	integerBounds := signedBounds(t, math.MinInt64, math.MaxInt64)
	integer := choice.OfInteger(integerBounds)
	recorded := choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(-3)}
	float := choice.OfFloat(floatBounds(t, -math.MaxFloat64, math.SmallestNonzeroFloat64, choice.AdmitNaN,
		choice.Width64))
	sequence := choice.OfSequence(choice.MustSequenceBounds(math.MaxUint32, sizes(t, 0, math.MaxInt64)))
	assert.MaxAllocs(t, func() { _ = integer.String() }, 1, "String allocates the text of integer bounds")
	assert.MaxAllocs(t, func() { _ = float.String() }, 1, "String allocates the text of float bounds with NaN")
	assert.MaxAllocs(t, func() { _ = sequence.String() }, 1, "String allocates the text of sequence bounds")
	assert.MaxAllocs(t, func() { _ = choice.OfInteger(integerBounds) }, 0, "OfInteger allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.OfFloat(choice.FloatBounds{}) }, 0, "OfFloat allocates nothing")
	assert.MaxAllocs(t, func() { _ = choice.OfSequence(choice.SequenceBounds{}) }, 0, "OfSequence allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Kind() }, 0, "Kind allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Integer() }, 0, "Integer allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Float() }, 0, "Float allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Sequence() }, 0, "Sequence allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Target() }, 0, "Target of integer bounds allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Admits(recorded) }, 0, "Admits allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Coerce(recorded) }, 0, "Coerce of an integer allocates nothing")
	assert.MaxAllocs(t, func() { _ = integer.Key(recorded) }, 0, "Key allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = integer.MarshalJSON() }, marshalAllocs, "MarshalJSON allocates its JSON")
}

// BenchmarkBounds measures the constructors and each operation of Bounds
// on integer bounds, which allocate nothing.
func BenchmarkBounds(b *testing.B) {
	integerBounds := signedBounds(b, math.MinInt64, math.MaxInt64)
	bounds := choice.OfInteger(integerBounds)
	recorded := choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(-3)}

	b.Run("OfInteger", func(b *testing.B) {
		var got choice.Bounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.OfInteger(integerBounds)
		}
		assert.Equal(b, got.Kind(), choice.Integer, "integer bounds")
	})

	b.Run("OfFloat", func(b *testing.B) {
		var got choice.Bounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.OfFloat(choice.FloatBounds{})
		}
		assert.Equal(b, got.Kind(), choice.Float, "float bounds")
	})

	b.Run("OfSequence", func(b *testing.B) {
		var got choice.Bounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = choice.OfSequence(choice.SequenceBounds{})
		}
		assert.Equal(b, got.Kind(), choice.Sequence, "sequence bounds")
	})

	b.Run("Kind", func(b *testing.B) {
		var got choice.Kind
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Kind()
		}
		assert.Equal(b, got, choice.Integer, "the kind")
	})

	b.Run("Integer", func(b *testing.B) {
		var got choice.IntegerBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Integer()
		}
		assert.Equal(b, got, integerBounds, "the integer bounds")
	})

	b.Run("Float", func(b *testing.B) {
		var got choice.FloatBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Float()
		}
		assert.Equal(b, got, choice.FloatBounds{}, "no float bounds")
	})

	b.Run("Sequence", func(b *testing.B) {
		var got choice.SequenceBounds
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Sequence()
		}
		assert.Equal(b, got, choice.SequenceBounds{}, "no sequence bounds")
	})

	b.Run("Target", func(b *testing.B) {
		var got choice.Choice
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Target()
		}
		assert.Equal(b, got.Integer, choice.Int{}, "zero")
	})

	b.Run("Admits", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Admits(recorded)
		}
		assert.True(b, got, "-3 is in the signed range")
	})

	b.Run("Coerce", func(b *testing.B) {
		var got choice.Choice
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Coerce(recorded)
		}
		assert.Equal(b, got.Integer, choice.IntOf(-3), "the recorded value")
	})

	b.Run("Key", func(b *testing.B) {
		var got choice.Key
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = bounds.Key(recorded)
		}
		assert.Equal(b, got.Compare(integerBounds.Key(choice.IntOf(-3))), 0, "the key of -3")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got = bounds.String()
		}
		assert.Equal(b, got, "integer in [-9223372036854775808, 9223372036854775807]", "the text")
	})

	b.Run("MarshalJSON", func(b *testing.B) {
		got, _ := bounds.MarshalJSON()
		c := bench.Start(b).MaxAllocs(marshalAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = bounds.MarshalJSON()
		}
		assert.Equal(b, string(got), `{"kind":"integer","min":"-9223372036854775808","max":"9223372036854775807"}`,
			"the corpus form")
	})
}
