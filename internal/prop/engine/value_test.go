// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// fields is a struct with unexported fields of each kind that the
// canonical form reads through: an integer, a pointer and an interface.
type fields struct {
	// n is an integer.
	n int
	// p is a pointer.
	p *int
	// v is an interface.
	v any
}

// TestValue checks the equality that a unique collection discards by: the
// same type and the same value, with floats compared by their bits.
func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("UniqueList", func(t *testing.T) {
		t.Parallel()

		channel := make(chan int)
		negativeZero := math.Copysign(0, -1)
		selfList, otherSelfList := []any{nil}, []any{nil}
		selfList[0], otherSelfList[0] = selfList, otherSelfList
		selfPointer := &fields{}
		selfPointer.v = selfPointer
		tests := []struct {
			name string
			a, b any
			want bool
		}{
			{name: "keeps -0 apart from +0", a: 0.0, b: negativeZero, want: true},
			{
				name: "keeps one of two NaNs with other payloads",
				a:    math.NaN(),
				b:    math.Float64frombits(0x7FF8000000000001),
			},
			{name: "keeps float32 values apart by their bits", a: float32(1.5), b: float32(2.5), want: true},
			{name: "keeps an int apart from an int64 of the same value", a: 1, b: int64(1), want: true},
			{name: "keeps an int apart from a uint of the same value", a: 1, b: uint(1), want: true},
			{name: "keeps a defined type apart from its underlying type", a: celsius(5), b: int8(5), want: true},
			{name: "keeps one of two equal integers", a: -7, b: -7},
			{name: "keeps negative integers apart", a: -1, b: -2, want: true},
			{name: "keeps unsigned integers apart", a: uint(1), b: uint(2), want: true},
			{name: "keeps true apart from false", a: true, b: false, want: true},
			{name: "keeps one of two trues", a: true, b: true},
			{name: "keeps strings apart by their text", a: "a", b: "b", want: true},
			{name: "keeps one of two equal strings", a: "ab", b: "ab"},
			{
				name: "keeps lists of texts apart by where each text ends",
				a:    []string{"ab", "c"},
				b:    []string{"a", "bc"},
				want: true,
			},
			{name: "keeps one of a nil slice and an empty slice", a: []int(nil), b: []int{}},
			{name: "keeps arrays apart by their elements", a: [2]int{1, 2}, b: [2]int{2, 1}, want: true},
			{
				name: "keeps one of two maps with the same entries",
				a:    map[string]int{"a": 1, "b": 2},
				b:    map[string]int{"b": 2, "a": 1},
			},
			{
				name: "keeps maps apart by their values",
				a:    map[string]int{"a": 1},
				b:    map[string]int{"a": 2},
				want: true,
			},
			{name: "keeps maps apart by their keys", a: map[string]int{"a": 1}, b: map[string]int{"b": 1}, want: true},
			{name: "keeps one of two pointers to equal values", a: new(5), b: new(5)},
			{name: "keeps a nil pointer apart from a pointer to the zero value", a: (*int)(nil), b: new(0), want: true},
			{name: "keeps structs apart by an unexported field", a: fields{n: 1}, b: fields{n: 2}, want: true},
			{
				name: "keeps one of two structs with equal fields",
				a:    fields{n: 1, p: new(2), v: "x"},
				b:    fields{n: 1, p: new(2), v: "x"},
			},
			{
				name: "keeps a nil interface field apart from one with the zero value",
				a:    fields{},
				b:    fields{v: 0},
				want: true,
			},
			{name: "keeps nil apart from the zero integer", a: nil, b: 0, want: true},
			{name: "keeps one of two nils", a: nil, b: nil},
			{
				name: "keeps complex numbers apart by the sign of a zero part",
				a:    complex(0, 0),
				b:    complex(0, negativeZero),
				want: true,
			},
			{name: "keeps one of two complex NaNs", a: complex(math.NaN(), 0), b: complex(math.NaN(), 0)},
			{name: "keeps two channels apart", a: make(chan int), b: make(chan int), want: true},
			{name: "keeps one of a channel stated twice", a: channel, b: channel},
			{name: "keeps one of two lists that contain themselves", a: selfList, b: otherSelfList},
			{
				name: "keeps a list that contains itself apart from a list that contains an empty list",
				a:    selfList,
				b:    []any{[]any{}},
				want: true,
			},
			{
				name: "keeps a pointer to a struct that contains it apart from a pointer to a zero struct",
				a:    selfPointer,
				b:    &fields{},
				want: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g := engine.UniqueList(engine.SampledFrom(tt.a, tt.b), unbounded(t, 0))
				got, _ := decode(t, g, integers(1, 0, 1, 1, 0)...)
				assert.Equal(t, len(got) == 2, tt.want, "whether the list keeps the second value")
			})
		}
	})
}
