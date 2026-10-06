// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package literal_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/literal"
)

// The allocation ceilings, measured once encoding/json has built its
// encoders for the literal's types.
const (
	// encodeAllocs are the allocations of Encode on a list of two integers.
	encodeAllocs = 15
	// opaqueAllocs are the allocations of Opaque.
	opaqueAllocs = 3
	// detailAllocs are the allocations of Detail on an error.
	detailAllocs = 4
	// plainAllocs are the allocations of Plain on an int.
	plainAllocs = 4
	// refusedAllocs are the allocations of Encode on a list of more parts
	// than the bound: its walk, and no literal of the list.
	refusedAllocs = 1
	// cycleAllocs are the allocations of Encode on a map that contains
	// itself, which ends at the level past the levels of a literal.
	cycleAllocs = 103
)

// faulty is a value whose text cannot be marshalled.
type faulty struct{}

// MarshalText returns an error.
func (faulty) MarshalText() ([]byte, error) {
	return nil, errors.New("literal_test: no text")
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

// TestEncode checks the typed literal of each kind of Go value, and the
// values that have none.
func TestEncode(t *testing.T) {
	t.Parallel()

	t.Run("Encode", func(t *testing.T) {
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
			{name: "returns an int for a *big.Int", give: big.NewInt(-3), want: `{"type":"int","value":-3}`},
			{name: "returns an int for a big.Int", give: *big.NewInt(5), want: `{"type":"int","value":5}`},
			{
				name: "returns a string for a *big.Int beyond 2^53 - 1",
				give: new(big.Int).Lsh(big.NewInt(1), 64),
				want: `{"type":"int","value":"18446744073709551616"}`,
			},
			{
				name: "returns a list of int for a list of *big.Int",
				give: []*big.Int{big.NewInt(1), new(big.Int).Lsh(big.NewInt(1), 64)},
				want: `{"type":"list","of":"int","value":[1,"18446744073709551616"]}`,
			},
			{name: "returns a float", give: 1.5, want: `{"type":"float","value":1.5}`},
			{name: "returns a float32", give: float32(0.5), want: `{"type":"float","value":0.5}`},
			{name: "returns NaN by its name", give: math.NaN(), want: `{"type":"float","value":"NaN"}`},
			{name: "returns Inf by its name", give: math.Inf(1), want: `{"type":"float","value":"Inf"}`},
			{name: "returns -Inf by its name", give: math.Inf(-1), want: `{"type":"float","value":"-Inf"}`},
			{name: "returns a string", give: "x", want: `{"type":"string","value":"x"}`},
			{name: "returns bytes in hexadecimal", give: []byte("hi"), want: `{"type":"bytes","value":"6869"}`},
			{name: "returns an array of bytes", give: [2]byte{1, 2}, want: `{"type":"bytes","value":"0102"}`},
			{name: "returns null for nil bytes", give: []byte(nil), want: `{"type":"null"}`},
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
			{
				name: "returns the absent list of its scalar type for a nil list",
				give: []string(nil),
				want: `{"type":"list","of":"string","value":null}`,
			},
			{
				name: "returns the absent list of strings for a nil list of a TextMarshaler",
				give: []time.Time(nil),
				want: `{"type":"list","of":"string","value":null}`,
			},
			{
				name: "returns the absent list of ints for a nil list of big.Int",
				give: []big.Int(nil),
				want: `{"type":"list","of":"int","value":null}`,
			},
			{
				name: "returns the absent list of floats for a nil list of float32",
				give: []float32(nil),
				want: `{"type":"list","of":"float","value":null}`,
			},
			{
				name: "returns null for a nil list of a type that is no scalar",
				give: []point(nil),
				want: `{"type":"null"}`,
			},
			{
				name: "returns null for a nil list of pointers, whose elements can be null",
				give: []*big.Int(nil),
				want: `{"type":"null"}`,
			},
			{
				name: "returns the absent map of its scalar type for a nil map from strings",
				give: map[string]int(nil),
				want: `{"type":"map","key":"string","of":"int","value":null}`,
			},
			{
				name: "returns null for a nil map whose keys are no strings",
				give: map[int]bool(nil),
				want: `{"type":"null"}`,
			},
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
				name: "returns a record of its fields in order",
				give: literal.Record{Fields: []literal.Field{{Name: "z", Value: 1}, {Name: "a", Value: nil}}},
				want: `{"type":"record","fields":[["z",{"type":"int","value":1}],["a",{"type":"null"}]]}`,
			},
			{
				name: "returns the entries of pairs in their order",
				give: literal.Pairs{Entries: []literal.Entry{{Key: 9, Value: "b"}, {Key: 10, Value: "a"}}},
				want: `{"type":"map","entries":[[{"type":"int","value":9},{"type":"string","value":"b"}],` +
					`[{"type":"int","value":10},{"type":"string","value":"a"}]]}`,
			},
			{
				name: "returns a variant without a payload",
				give: literal.Variant{Name: "pending"},
				want: `{"type":"variant","name":"pending"}`,
			},
			{
				name: "returns a variant whose payload is absent",
				give: literal.Variant{Name: "refunded", HasPayload: true},
				want: `{"type":"variant","name":"refunded","payload":{"type":"null"}}`,
			},
			{
				name: "returns a variant with its payload",
				give: &literal.Variant{Name: "paid", Payload: 5, HasPayload: true},
				want: `{"type":"variant","name":"paid","payload":{"type":"int","value":5}}`,
			},
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
			{
				name: "returns a record whose value nests 58 levels",
				give: literal.Record{Fields: []literal.Field{{Name: "k", Value: wrapped(28, []int{1})}}},
				want: "",
			},
			{
				name: "returns a variant whose payload nests 60 levels",
				give: literal.Variant{Name: "v", Payload: wrapped(29, []int{1}), HasPayload: true},
				want: "",
			},
			{name: "returns a list of 65,536 parts", give: make([]int, 65535), want: ""},
			{name: "returns a string of 65,536 parts", give: strings.Repeat("x", 65535), want: ""},
			{name: "returns bytes of 65,536 parts", give: make([]byte, 65535), want: ""},
			{name: "returns a text of 65,536 parts", give: longText(65535), want: ""},
			{
				name: "returns a record whose name has 65,534 parts",
				give: literal.Record{Fields: []literal.Field{{Name: strings.Repeat("x", 65534)}}},
				want: "",
			},
			{
				name: "returns a variant whose name has 65,535 parts",
				give: literal.Variant{Name: strings.Repeat("x", 65535)},
				want: "",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := literal.Encode(tt.give)
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
			{name: "returns false for a map of 32,768 entries, 65,537 parts or more", give: numbered(32768)},
			{
				name: "returns false for pairs of 32,768 entries, 65,537 parts or more",
				give: literal.Pairs{Entries: make([]literal.Entry, 32768)},
			},
			{
				name: "returns false for a record of 65,536 fields, 65,537 parts or more",
				give: literal.Record{Fields: make([]literal.Field, 65536)},
			},
			{name: "returns false for a string of 65,537 parts", give: strings.Repeat("x", 65536)},
			{name: "returns false for bytes of 65,537 parts", give: make([]byte, 65536)},
			{name: "returns false for a text of 65,537 parts", give: longText(65536)},
			{
				name: "returns false for a record field without a literal",
				give: literal.Record{Fields: []literal.Field{{Name: "c", Value: make(chan int)}}},
			},
			{
				name: "returns false for a record whose name has 65,536 parts",
				give: literal.Record{Fields: []literal.Field{{Name: strings.Repeat("x", 65536)}}},
			},
			{
				name: "returns false for a record whose name and value have 65,537 parts",
				give: literal.Record{Fields: []literal.Field{{Name: strings.Repeat("x", 65535)}}},
			},
			{
				name: "returns false for a record whose value nests 59 levels",
				give: literal.Record{Fields: []literal.Field{{Name: "k", Value: wrapped(29, nil)}}},
			},
			{
				name: "returns false for a pairs key without a literal",
				give: literal.Pairs{Entries: []literal.Entry{{Key: complex(1, 2), Value: 1}}},
			},
			{
				name: "returns false for a pairs value without a literal",
				give: literal.Pairs{Entries: []literal.Entry{{Key: 1, Value: complex(1, 2)}}},
			},
			{
				name: "returns false for a payload without a literal",
				give: literal.Variant{Name: "v", Payload: make(chan int), HasPayload: true},
			},
			{
				name: "returns false for a variant whose name has 65,536 parts",
				give: literal.Variant{Name: strings.Repeat("x", 65536)},
			},
			{
				name: "returns false for a variant whose payload nests 61 levels",
				give: literal.Variant{Name: "v", Payload: wrapped(30, nil), HasPayload: true},
			},
		}
		for _, tt := range none {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := literal.Encode(tt.give)
				assert.False(t, ok, "the value has no literal")
				assert.Nil(t, got, "no JSON")
			})
		}
	})

	t.Run("Plain", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give any
			want string
		}{
			{name: "returns a number for an int", give: -3, want: `-3`},
			{
				name: "returns a string for an integer beyond 2^53 - 1",
				give: uint64(math.MaxUint64),
				want: `"18446744073709551615"`,
			},
			{name: "returns a number for a float", give: 1.5, want: `1.5`},
			{name: "returns the name of NaN", give: math.NaN(), want: `"NaN"`},
			{name: "returns the name of -Inf", give: math.Inf(-1), want: `"-Inf"`},
			{name: "returns a bool", give: true, want: `true`},
			{name: "returns a string", give: "x", want: `"x"`},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, ok := literal.Plain(tt.give)
				assert.True(t, ok, "the value is a scalar")
				text, err := json.Marshal(got)
				assert.NoError(t, err, "the value is JSON")
				assert.Equal(t, string(text), tt.want, "the JSON value")
			})
		}

		for _, give := range []any{[]int{1}, complex(1, 2)} {
			t.Run(fmt.Sprintf("returns false for a %T", give), func(t *testing.T) {
				t.Parallel()
				got, ok := literal.Plain(give)
				assert.False(t, ok, "the value is no scalar")
				assert.Nil(t, got, "no value")
			})
		}
	})

	t.Run("Opaque", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{name: "returns the text", give: "func(int) bool", want: `{"type":"opaque","text":"func(int) bool"}`},
			{name: "escapes a quote and a newline", give: "a \"b\"\n", want: `{"type":"opaque","text":"a \"b\"\n"}`},
			{
				name: "returns the replacement character for a byte that is not UTF-8",
				give: "\xff",
				want: `{"type":"opaque","text":"�"}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, string(literal.Opaque(tt.give)), tt.want, "the literal")
			})
		}
	})

	t.Run("Detail", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give any
			want string
		}{
			{
				name: "returns the literal that Encode returns",
				give: []int{1, 2},
				want: `{"type":"list","of":"int","value":[1,2]}`,
			},
			{name: "returns null for nil", give: nil, want: `{"type":"null"}`},
			{
				name: "returns an error as opaque",
				give: errors.New("literal_test: boom"),
				want: `{"type":"opaque","text":"literal_test: boom"}`,
			},
			{
				name: "returns a context as opaque",
				give: context.Background(),
				want: `{"type":"opaque","text":"context.Background"}`,
			},
			{
				name: "returns a value without a literal as opaque",
				give: complex(1, 2),
				want: `{"type":"opaque","text":"(1+2i)"}`,
			},
			{
				name: "returns a value that contains itself as opaque with the cycle marked",
				give: selfContaining(),
				want: string(literal.Opaque("map[self:<cycle>]")),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, string(literal.Detail(tt.give)), tt.want, "the literal")
			})
		}
	})
}

// TestEncodeAllocs checks the ceilings of Encode on a list of two
// integers, of Opaque, of Detail on an error, and of Plain on an int.
func TestEncodeAllocs(t *testing.T) {
	values := []int{1, 2}
	boom := errors.New("literal_test: boom")
	var number any = 1234
	_ = literal.Opaque("warm")
	assert.MaxAllocs(t, func() { _, _ = literal.Encode(values) }, encodeAllocs, "Encode allocates its JSON")
	assert.MaxAllocs(t, func() { _ = literal.Opaque("func(int) bool") }, opaqueAllocs, "Opaque allocates its JSON")
	assert.MaxAllocs(t, func() { _ = literal.Detail(boom) }, detailAllocs, "Detail allocates the text and its JSON")
	assert.MaxAllocs(t, func() { _, _ = literal.Plain(number) }, plainAllocs, "Plain allocates the value")
	var huge any = make([]int, 65536)
	assert.MaxAllocs(t, func() { _, _ = literal.Encode(huge) }, refusedAllocs,
		"Encode refuses a list of more parts than the bound before it allocates the list's literal")
	var looped any = selfContaining()
	assert.MaxAllocs(t, func() { _, _ = literal.Encode(looped) }, cycleAllocs,
		"Encode refuses a map that contains itself once it nests past the levels of a literal")
}

// BenchmarkEncode measures Encode on a list of two integers, Opaque,
// Detail on an error, and Plain on an int.
func BenchmarkEncode(b *testing.B) {
	values := []int{1, 2}
	boom := errors.New("literal_test: boom")
	var number any = 1234

	b.Run("Plain", func(b *testing.B) {
		got, _ := literal.Plain(number)
		c := bench.Start(b).MaxAllocs(plainAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = literal.Plain(number)
		}
		assert.Equal(b, got, any(json.Number("1234")), "the value")
	})

	b.Run("Encode", func(b *testing.B) {
		// The first call builds encoding/json's encoders for the literal's
		// types, which a short run would count against the ceiling.
		got, _ := literal.Encode(values)
		c := bench.Start(b).MaxAllocs(encodeAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = literal.Encode(values)
		}
		assert.Equal(b, string(got), `{"type":"list","of":"int","value":[1,2]}`, "the literal")
	})

	b.Run("Opaque", func(b *testing.B) {
		got := literal.Opaque("func(int) bool")
		c := bench.Start(b).MaxAllocs(opaqueAllocs)
		defer c.End()
		for c.Loop() {
			got = literal.Opaque("func(int) bool")
		}
		assert.Equal(b, string(got), `{"type":"opaque","text":"func(int) bool"}`, "the literal")
	})

	b.Run("Detail", func(b *testing.B) {
		got := literal.Detail(boom)
		c := bench.Start(b).MaxAllocs(detailAllocs)
		defer c.End()
		for c.Loop() {
			got = literal.Detail(boom)
		}
		assert.Equal(b, string(got), `{"type":"opaque","text":"literal_test: boom"}`, "the literal")
	})
}

// wrapped returns inner inside count lists of one element. Each list adds
// two levels to the literal, and the innermost list of one scalar nests two
// levels itself.
func wrapped(count int, inner any) any {
	node := inner
	for range count {
		node = []any{node}
	}
	return node
}

// numbered returns a map of the integers from 0 to n - 1, each to itself.
func numbered(n int) map[int]int {
	m := make(map[int]int, n)
	for i := range n {
		m[i] = i
	}
	return m
}

// selfContaining returns a map whose one entry is the map itself.
func selfContaining() map[string]any {
	m := map[string]any{}
	m["self"] = m
	return m
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
