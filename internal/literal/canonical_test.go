// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal_test

import (
	"math"
	"math/big"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/literal"
)

// canonicalAllocs are the allocations of Canonical on a list of two
// integers: the builder's growth and the formatting of each part.
const canonicalAllocs = 5

// TestCanonical checks the text of each kind of value under the
// definition's equality.
func TestCanonical(t *testing.T) {
	t.Parallel()

	t.Run("Canonical", func(t *testing.T) {
		t.Parallel()

		seven := 7
		var noInt *int
		tests := []struct {
			name string
			give any
			want string
		}{
			{name: "returns null for nil", give: nil, want: "null"},
			{name: "returns null for a nil pointer", give: noInt, want: "null"},
			{name: "returns a bool", give: true, want: "bool:true"},
			{name: "returns an int", give: 7, want: "int:7"},
			{name: "returns an int64 as an int", give: int64(-7), want: "int:-7"},
			{name: "returns an unsigned integer as an int", give: uint8(7), want: "int:7"},
			{name: "returns the integer of a defined type as an int", give: celsius(21), want: "int:21"},
			{
				name: "returns a *big.Int as an int",
				give: new(big.Int).Lsh(big.NewInt(1), 64),
				want: "int:18446744073709551616",
			},
			{name: "returns a big.Int as an int", give: *big.NewInt(-3), want: "int:-3"},
			{name: "returns a float", give: 1.5, want: "float:1.5"},
			{name: "returns a float32 as a float", give: float32(1.5), want: "float:1.5"},
			{name: "returns negative zero apart from zero", give: math.Copysign(0, -1), want: "float:-0"},
			{name: "returns zero", give: 0.0, want: "float:0"},
			{name: "returns every NaN as one value", give: math.Float64frombits(0x7ff8000000000001), want: "float:NaN"},
			{name: "returns a quoted string", give: `a"b`, want: `string:"a\"b"`},
			{name: "returns bytes in hexadecimal", give: []byte{1, 2}, want: "bytes:0102"},
			{name: "returns a list of its elements' texts", give: []int{1, 2}, want: "list:[int:1,int:2]"},
			{
				name: "returns a list of any as the list of its elements",
				give: []any{int64(1), 2},
				want: "list:[int:1,int:2]",
			},
			{
				name: "returns a map's entries sorted by their keys' texts",
				give: map[string]int{"b": 2, "a": 1},
				want: `map:{string:"a"=int:1,string:"b"=int:2}`,
			},
			{
				name: "returns pairs as the map of their entries",
				give: literal.Pairs{Entries: []literal.Entry{{Key: "b", Value: int64(2)}, {Key: "a", Value: 1}}},
				want: `map:{string:"a"=int:1,string:"b"=int:2}`,
			},
			{
				name: "returns a record's fields in order",
				give: literal.Record{Fields: []literal.Field{{Name: "z", Value: 1}, {Name: "a", Value: nil}}},
				want: `record:["z"=int:1,"a"=null]`,
			},
			{
				name: "returns a variant without a payload",
				give: literal.Variant{Name: "pending"},
				want: `variant:"pending"`,
			},
			{
				name: "returns a variant whose payload is absent",
				give: literal.Variant{Name: "refunded", HasPayload: true},
				want: `variant:"refunded"(null)`,
			},
			{
				name: "returns a variant with its payload",
				give: &literal.Variant{Name: "paid", Payload: uint16(5), HasPayload: true},
				want: `variant:"paid"(int:5)`,
			},
			{name: "returns the target of a pointer", give: &seven, want: "int:7"},
			{name: "returns the empty text for a channel", give: make(chan int), want: ""},
			{name: "returns the empty text for another struct", give: point{X: 1}, want: ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, literal.Canonical(tt.give), tt.want, "the canonical text")
			})
		}
	})
}

// TestCanonicalAllocs checks the ceiling of Canonical on a list of two
// integers.
func TestCanonicalAllocs(t *testing.T) {
	values := []int{1, 2}
	assert.MaxAllocs(t, func() { _ = literal.Canonical(values) }, canonicalAllocs, "Canonical allocates its text")
}

// BenchmarkCanonical measures Canonical on a list of two integers.
func BenchmarkCanonical(b *testing.B) {
	values := []int{1, 2}

	b.Run("Canonical", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(canonicalAllocs)
		defer c.End()
		for c.Loop() {
			got = literal.Canonical(values)
		}
		assert.Equal(b, got, "list:[int:1,int:2]", "the text")
	})
}
