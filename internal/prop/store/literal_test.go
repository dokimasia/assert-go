// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package store_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/store"
)

// literalAllocs are the allocations of Literal on a list of two integers,
// measured once encoding/json has built its encoders for the literal's
// types.
const literalAllocs = 15

// celsius is a defined type over an integer, whose literal is the integer.
type celsius int

// point is a struct with exported and unexported fields.
type point struct {
	// X is exported, so the literal states it.
	X int
	// label is unexported, so the literal leaves it out.
	label string
	// Tags are exported, so the literal states them.
	Tags []bool
}

// faulty is a value whose text cannot be marshalled.
type faulty struct{}

// MarshalText returns an error.
func (faulty) MarshalText() ([]byte, error) {
	return nil, errors.New("store_test: no text")
}

// garbled is a value whose text is not UTF-8.
type garbled struct{}

// MarshalText returns a byte that is not UTF-8.
func (garbled) MarshalText() ([]byte, error) {
	return []byte{0xff}, nil
}

// cycle is a struct that can contain itself.
type cycle struct {
	// Next is the struct that follows.
	Next *cycle
}

// longText is a value whose text has a stated number of bytes.
type longText int

// MarshalText returns as many bytes as the value states.
func (n longText) MarshalText() ([]byte, error) {
	return []byte(strings.Repeat("x", int(n))), nil
}

// TestLiteral checks the typed literal of each kind of Go value, and the
// values that have none.
func TestLiteral(t *testing.T) {
	t.Parallel()

	t.Run("Literal", func(t *testing.T) {
		t.Parallel()

		seven := 7
		var noInt *int
		looped := &cycle{}
		looped.Next = looped
		selfList := []any{nil}
		selfList[0] = selfList
		var selfPointer any
		selfPointer = &selfPointer
		tests := []struct {
			name string
			give any
			want string
		}{
			{name: "returns null for nil", give: nil, want: `{"type":"null"}`},
			{name: "returns null for a nil pointer", give: noInt, want: `{"type":"null"}`},
			{name: "returns a bool", give: true, want: `{"type":"bool","value":true}`},
			{name: "returns an int", give: -3, want: `{"type":"int","value":-3}`},
			{name: "returns the integer of a defined type", give: celsius(21), want: `{"type":"int","value":21}`},
			{
				name: "returns a number for 2^53 - 1",
				give: int64(9007199254740991),
				want: `{"type":"int","value":9007199254740991}`,
			},
			{
				name: "returns a string for 2^53",
				give: int64(9007199254740992),
				want: `{"type":"int","value":"9007199254740992"}`,
			},
			{
				name: "returns a number for -(2^53 - 1)",
				give: int64(-9007199254740991),
				want: `{"type":"int","value":-9007199254740991}`,
			},
			{
				name: "returns a string for -2^53",
				give: int64(-9007199254740992),
				want: `{"type":"int","value":"-9007199254740992"}`,
			},
			{name: "returns an unsigned integer", give: uint8(200), want: `{"type":"int","value":200}`},
			{
				name: "returns a number for an unsigned 2^53 - 1",
				give: uint64(9007199254740991),
				want: `{"type":"int","value":9007199254740991}`,
			},
			{
				name: "returns a string for the largest uint64",
				give: uint64(math.MaxUint64),
				want: `{"type":"int","value":"18446744073709551615"}`,
			},
			{name: "returns a float", give: 1.5, want: `{"type":"float","value":1.5}`},
			{name: "returns a float32", give: float32(0.5), want: `{"type":"float","value":0.5}`},
			{name: "returns NaN by its name", give: math.NaN(), want: `{"type":"float","value":"NaN"}`},
			{name: "returns Inf by its name", give: math.Inf(1), want: `{"type":"float","value":"Inf"}`},
			{name: "returns -Inf by its name", give: math.Inf(-1), want: `{"type":"float","value":"-Inf"}`},
			{name: "returns a string", give: "x", want: `{"type":"string","value":"x"}`},
			{name: "returns bytes in hexadecimal", give: []byte("hi"), want: `{"type":"bytes","value":"6869"}`},
			{name: "returns an array of bytes", give: [2]byte{1, 2}, want: `{"type":"bytes","value":"0102"}`},
			{name: "returns no bytes for nil bytes", give: []byte(nil), want: `{"type":"bytes","value":""}`},
			{
				name: "returns a list of one scalar type",
				give: []int{1, 2},
				want: `{"type":"list","of":"int","value":[1,2]}`,
			},
			{
				name: "returns the names of NaN in a list of floats",
				give: []float64{math.NaN(), 2},
				want: `{"type":"list","of":"float","value":["NaN",2]}`,
			},
			{
				name: "returns a string for a large integer in a list",
				give: []uint64{math.MaxUint64},
				want: `{"type":"list","of":"int","value":["18446744073709551615"]}`,
			},
			{name: "returns items for an empty list", give: []int{}, want: `{"type":"list","items":[]}`},
			{name: "returns items for a nil list", give: []string(nil), want: `{"type":"list","items":[]}`},
			{
				name: "returns items for a list of two scalar types",
				give: []any{true, 1},
				want: `{"type":"list","items":[{"type":"bool","value":true},{"type":"int","value":1}]}`,
			},
			{
				name: "returns items for a list whose second element is no scalar",
				give: []any{1, []int{2}},
				want: `{"type":"list","items":[{"type":"int","value":1},{"type":"list","of":"int","value":[2]}]}`,
			},
			{
				name: "returns items for a list of byte strings",
				give: [][]byte{{1}},
				want: `{"type":"list","items":[{"type":"bytes","value":"01"}]}`,
			},
			{
				name: "returns null for a nil element",
				give: []any{nil},
				want: `{"type":"list","items":[{"type":"null"}]}`,
			},
			{
				name: "returns a map's entries sorted by the JSON of their keys",
				give: map[int]bool{9: false, 10: true},
				want: `{"type":"map","entries":[[{"type":"int","value":10},{"type":"bool","value":true}],` +
					`[{"type":"int","value":9},{"type":"bool","value":false}]]}`,
			},
			{name: "returns no entries for an empty map", give: map[string]int{}, want: `{"type":"map","entries":[]}`},
			{
				name: "returns a struct as a map of its exported fields in order",
				give: point{X: 1, label: "hidden", Tags: []bool{true}},
				want: `{"type":"map","entries":[[{"type":"string","value":"X"},{"type":"int","value":1}],` +
					`[{"type":"string","value":"Tags"},{"type":"list","of":"bool","value":[true]}]]}`,
			},
			{
				name: "returns the text of a TextMarshaler",
				give: time.Date(2026, time.October, 1, 9, 30, 0, 0, time.UTC),
				want: `{"type":"string","value":"2026-10-01T09:30:00Z"}`,
			},
			{name: "returns the target of a pointer", give: &seven, want: `{"type":"int","value":7}`},
			{
				name: "returns a map nested 61 levels",
				give: map[string]any{"k": wrapped(28, []int{1})},
				want: "",
			},
			{
				name: "returns lists nested 61 levels",
				give: wrapped(30, nil),
				want: "",
			},
			{
				name: "returns a map whose key nests 58 levels",
				give: deepKey(29),
				want: "",
			},
			{name: "returns a list of 65,536 parts", give: make([]int, 65535), want: ""},
			{name: "returns a string of 65,536 parts", give: strings.Repeat("x", 65535), want: ""},
			{name: "returns bytes of 65,536 parts", give: make([]byte, 65535), want: ""},
			{name: "returns a text of 65,536 parts", give: longText(65535), want: ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := store.Literal(tt.give)
				assert.True(t, ok, "the value has a literal")
				if tt.want != "" {
					assert.Equal(t, string(got), tt.want, "the literal")
				}
			})
		}

		none := []struct {
			name string
			give any
		}{
			{name: "returns false for a string that is not UTF-8", give: "\xff"},
			{name: "returns false for a complex number", give: complex(1, 2)},
			{name: "returns false for a channel", give: make(chan int)},
			{name: "returns false for a function", give: func() {}},
			{name: "returns false for a list with a function", give: []any{1, func() {}}},
			{name: "returns false for a map key without a literal", give: map[complex64]int{1: 1}},
			{name: "returns false for a map value without a literal", give: map[int]any{1: make(chan int)}},
			{name: "returns false for a struct field without a literal", give: struct{ C chan int }{}},
			{name: "returns false for a TextMarshaler that fails", give: faulty{}},
			{name: "returns false for a TextMarshaler whose text is not UTF-8", give: garbled{}},
			{name: "returns false for a list nested 62 levels", give: wrapped(30, []any{})},
			{name: "returns false for a list whose values are past level 61", give: wrapped(30, []int{1})},
			{name: "returns false for a value that contains itself", give: looped},
			{name: "returns false for a list that contains itself", give: selfList},
			{name: "returns false for a pointer to itself", give: selfPointer},
			{name: "returns false for a map whose key nests 60 levels", give: deepKey(30)},
			{name: "returns false for a map nested 62 levels", give: map[string]any{"k": wrapped(29, nil)}},
			{name: "returns false for a list of 65,537 parts", give: make([]int, 65536)},
			{name: "returns false for a string of 65,537 parts", give: strings.Repeat("x", 65536)},
			{name: "returns false for bytes of 65,537 parts", give: make([]byte, 65536)},
			{name: "returns false for a text of 65,537 parts", give: longText(65536)},
		}
		for _, tt := range none {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := store.Literal(tt.give)
				assert.False(t, ok, "the value has no literal")
				assert.Nil(t, got, "no JSON")
			})
		}
	})
}

// TestLiteralZeroAlloc checks the ceiling of Literal on a list of two
// integers.
func TestLiteralZeroAlloc(t *testing.T) {
	values := []int{1, 2}
	assert.MaxAllocs(t, func() { _, _ = store.Literal(values) }, literalAllocs, "Literal allocates its JSON")
}

// BenchmarkLiteral measures Literal on a list of two integers.
func BenchmarkLiteral(b *testing.B) {
	values := []int{1, 2}

	b.Run("Literal", func(b *testing.B) {
		// The first call builds encoding/json's encoders for the literal's
		// types, which a short run would count against the ceiling.
		got, _ := store.Literal(values)
		c := bench.Start(b).MaxAllocs(literalAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = store.Literal(values)
		}
		assert.Equal(b, string(got), `{"type":"list","of":"int","value":[1,2]}`, "the literal")
	})
}

// wrapped returns inner inside count lists of one element. Each list adds
// two levels to the literal, and the innermost list of one scalar nests
// two levels itself.
func wrapped(count int, inner any) any {
	node := inner
	for range count {
		node = []any{node}
	}
	return node
}

// deepKey returns a map with one entry, whose key is an integer inside
// levels arrays of one element and whose value is 1. The key's literal
// nests twice as many levels as there are arrays.
func deepKey(levels int) any {
	key := reflect.TypeFor[int]()
	for range levels {
		key = reflect.ArrayOf(1, key)
	}
	m := reflect.MakeMap(reflect.MapOf(key, reflect.TypeFor[int]()))
	m.SetMapIndex(reflect.New(key).Elem(), reflect.ValueOf(1))
	return m.Interface()
}
