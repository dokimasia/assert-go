// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package fault_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/fault"
)

// The allocations of the functions of a path, measured.
const (
	// stringAllocs are the allocations of String for a path whose keys
	// have texts of 64 bytes or fewer: the text it returns.
	stringAllocs = 1
	// longKeyAllocs are the allocations of String for a path with one key
	// whose text is longer than 64 bytes: the text it returns, and the
	// key's text formatted twice.
	longKeyAllocs = 3
)

// longKey is a key whose quoted text is longer than 64 bytes.
var longKey = strings.Repeat("a", 70)

// pathOfEvery is a path with a segment of every kind.
var pathOfEvery = fault.Path{
	fault.Field("order"), fault.Field("Lines"), fault.Index(120), fault.Key("a"), fault.Key(7),
	fault.Element(), fault.Variant("paid"),
}

// TestPath checks which segments are equal, and the selector that a path
// renders.
func TestPath(t *testing.T) {
	t.Parallel()

	t.Run("Field", func(t *testing.T) {
		t.Parallel()

		t.Run("returns equal segments for one name and different segments for two", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, fault.Field("order"), fault.Field("order"), "one name")
			assert.NotEqual(t, fault.Field("order"), fault.Field("lines"), "two names")
		})
	})

	t.Run("Index", func(t *testing.T) {
		t.Parallel()

		t.Run("returns equal segments for one index and a segment unlike a field", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, fault.Index(3), fault.Index(3), "one index")
			assert.NotEqual(t, fault.Index(3), fault.Index(4), "two indices")
			assert.NotEqual(t, fault.Index(0), fault.Field(""), "an index and a field")
		})
	})

	t.Run("Key", func(t *testing.T) {
		t.Parallel()

		t.Run("returns equal segments for one key and different segments for two", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, fault.Key("a"), fault.Key("a"), "one key")
			assert.NotEqual(t, fault.Key("a"), fault.Key("b"), "two keys")
		})
	})

	t.Run("Element", func(t *testing.T) {
		t.Parallel()

		t.Run("returns equal segments, unlike the first index", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, fault.Element(), fault.Element(), "every element")
			assert.NotEqual(t, fault.Element(), fault.Index(0), "every element and the first index")
		})
	})

	t.Run("Variant", func(t *testing.T) {
		t.Parallel()

		t.Run("returns equal segments for one name, unlike a field of the name", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, fault.Variant("paid"), fault.Variant("paid"), "one variant")
			assert.NotEqual(t, fault.Variant("paid"), fault.Field("paid"), "a variant and a field")
		})
	})

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give fault.Path
			want string
		}{
			{
				name: "returns the empty string for the empty path",
				give: nil,
				want: "",
			},
			{
				name: "returns a field at the front without a dot and every later field after one",
				give: fault.Path{fault.Field("order"), fault.Field("Lines")},
				want: "order.Lines",
			},
			{
				name: "returns an index in brackets in decimal",
				give: fault.Path{fault.Field("cases"), fault.Index(1234), fault.Index(-1)},
				want: "cases[1234][-1]",
			},
			{
				name: "returns a key of text quoted in brackets",
				give: fault.Path{fault.Field("counts"), fault.Key(`a"b`)},
				want: `counts["a\"b"]`,
			},
			{
				name: "returns a key of another type in brackets as fmt prints it",
				give: fault.Path{fault.Field("counts"), fault.Key(7), fault.Key(true)},
				want: "counts[7][true]",
			},
			{
				name: "returns a key whose text is longer than 64 bytes in full",
				give: fault.Path{fault.Field("counts"), fault.Key(longKey)},
				want: `counts["` + longKey + `"]`,
			},
			{
				name: "returns an element as a pair of empty brackets",
				give: fault.Path{fault.Field("order"), fault.Field("Lines"), fault.Element(), fault.Field("Note")},
				want: "order.Lines[].Note",
			},
			{
				name: "returns a variant in parentheses after a dot",
				give: fault.Path{fault.Field("payment"), fault.Variant("paid")},
				want: "payment.(paid)",
			},
			{
				name: "returns an index at the front without a separator",
				give: fault.Path{fault.Index(0), fault.Field("detail")},
				want: "[0].detail",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give.String(), tt.want, "the selector")
			})
		}
	})
}

// TestPathAllocs checks the allocation ceilings of the functions of a
// path.
func TestPathAllocs(t *testing.T) {
	long := fault.Path{fault.Field("counts"), fault.Key(longKey)}
	assert.MaxAllocs(t, func() { _ = fault.Field("order") }, 0, "Field allocates nothing")
	assert.MaxAllocs(t, func() { _ = fault.Index(120) }, 0, "Index allocates nothing")
	assert.MaxAllocs(t, func() { _ = fault.Key("a") }, 0, "Key allocates nothing")
	assert.MaxAllocs(t, func() { _ = fault.Element() }, 0, "Element allocates nothing")
	assert.MaxAllocs(t, func() { _ = fault.Variant("paid") }, 0, "Variant allocates nothing")
	assert.MaxAllocs(t, func() { _ = fault.Path(nil).String() }, 0, "String allocates nothing for the empty path")
	assert.MaxAllocs(t, func() { _ = pathOfEvery.String() }, stringAllocs, "String allocates the text it returns")
	assert.MaxAllocs(t, func() { _ = long.String() }, longKeyAllocs,
		"String allocates its text and formats a long key twice")
}

// BenchmarkPath measures each function of a path.
func BenchmarkPath(b *testing.B) {
	b.Run("Field", func(b *testing.B) {
		var got fault.Segment
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = fault.Field("order")
		}
		assert.Equal(b, got, fault.Field("order"), "the segment")
	})

	b.Run("Index", func(b *testing.B) {
		var got fault.Segment
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = fault.Index(120)
		}
		assert.Equal(b, got, fault.Index(120), "the segment")
	})

	b.Run("Key", func(b *testing.B) {
		var got fault.Segment
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = fault.Key("a")
		}
		assert.Equal(b, got, fault.Key("a"), "the segment")
	})

	b.Run("Element", func(b *testing.B) {
		var got fault.Segment
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = fault.Element()
		}
		assert.Equal(b, got, fault.Element(), "the segment")
	})

	b.Run("Variant", func(b *testing.B) {
		var got fault.Segment
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = fault.Variant("paid")
		}
		assert.Equal(b, got, fault.Variant("paid"), "the segment")
	})

	b.Run("String", func(b *testing.B) {
		var got string
		c := bench.Start(b).MaxAllocs(stringAllocs)
		defer c.End()
		for c.Loop() {
			got = pathOfEvery.String()
		}
		assert.Equal(b, got, `order.Lines[120]["a"][7][].(paid)`, "the selector")
	})
}
