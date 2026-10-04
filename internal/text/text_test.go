// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package text_test

import (
	"fmt"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/text"
)

// The allocations of Sprintf, measured.
const (
	// scalarAllocs are the allocations of a format with scalar arguments:
	// the text that fmt.Sprintf returns.
	scalarAllocs = 1
	// structAllocs are the allocations of a format with a struct of a map
	// and a slice of scalars, whose walk allocates nothing: what fmt
	// allocates for the struct, its map and its text.
	structAllocs = 6
	// fprintfAllocs are the allocations of Fprintf of scalars into a reset
	// builder: the builder's storage.
	fprintfAllocs = 1
)

// sink receives the text that a measured call returns.
var sink string

// account is a value without a cycle, of a struct, a map and a slice.
type account struct {
	Name   string
	Limits map[string]int
	Tags   []string
}

// node is a value that contains itself through a slice.
type node struct {
	Name     string
	Children []any
}

// link is a value that points at itself.
type link struct {
	Next  *link
	Items []any
}

// wide is a struct whose first field has more than 65,536 parts.
type wide struct {
	Values [70000]uint8
	Last   int
}

// hidden contains a scalar of each kind in unexported fields, and a map
// that contains itself.
type hidden struct {
	b  bool
	i  int8
	u  uint16
	f  float32
	c  complex64
	s  string
	ch chan int
	fn func()
	m  map[string]any
}

// TestText checks that Sprintf writes what fmt writes for a value that it
// walks to the end, and the structural text of any other value.
func TestText(t *testing.T) {
	t.Parallel()

	t.Run("Sprintf", func(t *testing.T) {
		t.Parallel()

		t.Run("returns what fmt.Sprintf returns for scalar arguments", func(t *testing.T) {
			t.Parallel()
			got := text.Sprintf("the key %s states %d values, %q", "min", 3, "of 4")
			assert.Equal(t, got, fmt.Sprintf("the key %s states %d values, %q", "min", 3, "of 4"), "fmt's text")
		})

		t.Run("returns what fmt.Sprintf returns for a value without a cycle, under each verb", func(t *testing.T) {
			t.Parallel()
			value := account{Name: "ada", Limits: map[string]int{"day": 3, "month": 9}, Tags: []string{"a", "b"}}
			for _, format := range []string{"%v", "%+v", "%#v", "%q"} {
				assert.Equal(t, text.Sprintf(format, value), fmt.Sprintf(format, value), "fmt's text under "+format)
			}
			pointer := &value
			assert.Equal(t, text.Sprintf("%+v", pointer), fmt.Sprintf("%+v", pointer), "fmt's text of a pointer")
		})

		t.Run("writes a map inside itself with the cycle marked", func(t *testing.T) {
			t.Parallel()
			m := map[string]any{"n": 1}
			m["self"] = m
			assert.Equal(t, text.Sprintf("%v", m), "map[n:1 self:<cycle>]", "the entries sorted by key")
		})

		t.Run("writes a slice inside itself with the cycle marked whatever the verb", func(t *testing.T) {
			t.Parallel()
			s := []any{nil, 2}
			s[0] = s
			assert.Equal(t, text.Sprintf("%d values", s), "[<cycle> 2] values", "the elements in order")
		})

		t.Run("writes a struct's fields with their names and its pointer after an ampersand", func(t *testing.T) {
			t.Parallel()
			root := &node{Name: "root", Children: []any{nil}}
			root.Children[0] = root.Children
			assert.Equal(t, text.Sprintf("%v", root), "&{Name:root Children:[<cycle>]}", "the layout of %+v")
		})

		t.Run("writes a pointer below the top level as its address", func(t *testing.T) {
			t.Parallel()
			l := &link{Items: []any{nil}}
			l.Next, l.Items[0] = l, l.Items
			assert.Matches(t, text.Sprintf("%v", l), `^&\{Next:0x[0-9a-f]+ Items:\[<cycle>\]\}$`,
				"the address of the target and the cycle of the slice")
		})

		t.Run("writes the scalars of unexported fields as fmt writes them", func(t *testing.T) {
			t.Parallel()
			h := hidden{
				b: true, i: -3, u: 7, f: 1.5, c: complex(1, 2), s: "x",
				ch: make(chan int), fn: nil, m: map[string]any{},
			}
			h.m["self"] = h.m
			assert.Matches(t, text.Sprintf("%v", h),
				`^\{b:true i:-3 u:7 f:1\.5 c:\(1\+2i\) s:x ch:0x[0-9a-f]+ fn:<nil> m:map\[self:<cycle>\]\}$`,
				"each scalar, the address of a channel, and nil for a nil function")
		})

		t.Run("cuts a value after its 65,536th part", func(t *testing.T) {
			t.Parallel()
			got := text.Sprintf("%v", make([]int, 70000))
			assert.Equal(t, strings.Count(got, "0"), 65535, "the slice and 65,535 elements are the parts written")
			assert.HasSuffix(t, got, " …]", "a mark in place of the rest")
		})

		t.Run("cuts a struct after its 65,536th part and writes no field after the cut", func(t *testing.T) {
			t.Parallel()
			got := text.Sprintf("%v", wide{Last: 1})
			assert.Equal(t, strings.Count(got, "0"), 65534,
				"the struct, the array and 65,534 elements are the parts written")
			assert.HasSuffix(t, got, " …]}", "a mark in place of the rest, and no field Last")
		})

		t.Run("cuts a map in a key after its 65,536th part and writes no value after the cut", func(t *testing.T) {
			t.Parallel()
			got := text.Sprintf("%v", map[[70000]uint8]int{{}: 1})
			assert.Equal(t, strings.Count(got, "0"), 65534,
				"the map, the key and 65,534 of its elements are the parts written")
			assert.HasSuffix(t, got, " …]:]", "one mark in place of the rest, and no value")
		})
	})

	t.Run("Fprintf", func(t *testing.T) {
		t.Parallel()

		t.Run("writes what Sprintf returns after what the builder holds", func(t *testing.T) {
			t.Parallel()
			m := map[string]any{"n": 1}
			m["self"] = m
			var b strings.Builder
			b.WriteString("detail: ")
			text.Fprintf(&b, "%s %v, %d", "got", m, 3)
			assert.Equal(t, b.String(), "detail: got map[n:1 self:<cycle>], 3", "the text appended")
		})
	})
}

// TestTextAllocs checks the allocation ceilings of Sprintf.
func TestTextAllocs(t *testing.T) {
	value := account{Name: "ada", Limits: map[string]int{"day": 3}, Tags: []string{"a"}}
	assert.MaxAllocs(t, func() { sink = text.Sprintf("the key %s states %d values", "min", 3) }, scalarAllocs,
		"Sprintf of scalars allocates fmt's text")
	assert.MaxAllocs(t, func() { sink = text.Sprintf("%v", value) }, structAllocs,
		"Sprintf of a struct allocates its walk and fmt's text")
	var b strings.Builder
	b.Grow(64)
	assert.MaxAllocs(t, func() { b.Reset(); b.Grow(64); text.Fprintf(&b, "the key %s states %d values", "min", 3) },
		fprintfAllocs, "Fprintf of scalars allocates the builder's storage")
}

// BenchmarkText measures Sprintf.
func BenchmarkText(b *testing.B) {
	value := account{Name: "ada", Limits: map[string]int{"day": 3}, Tags: []string{"a"}}

	b.Run("Sprintf", func(b *testing.B) {
		c := bench.Start(b).MaxAllocs(structAllocs)
		defer c.End()
		for c.Loop() {
			sink = text.Sprintf("%v", value)
		}
		assert.Equal(b, sink, "{ada map[day:3] [a]}", "fmt's text")
	})

	b.Run("Fprintf", func(b *testing.B) {
		var out strings.Builder
		c := bench.Start(b).MaxAllocs(fprintfAllocs)
		defer c.End()
		for c.Loop() {
			out.Reset()
			out.Grow(64)
			text.Fprintf(&out, "the key %s states %d values", "min", 3)
		}
		assert.Equal(b, out.String(), "the key min states 3 values", "fmt's text")
	})
}
