// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package fault

import (
	"fmt"
	"strconv"
	"strings"
)

// step is the kind of a [Segment].
type step uint8

// The kinds of a segment.
const (
	// field is a field of a record, or the name of the input at the front
	// of a path.
	field step = iota
	// index is an item of a sequence.
	index
	// key is the entry of a map at a key.
	key
	// element is every element of a sequence type.
	element
	// variant is a variant of an enum.
	variant
)

// Segment is one step of a [Path]: a field, an index, a key, the element of
// a sequence type, or a variant. Two segments are equal when they state the
// same step.
type Segment struct {
	// step is the kind of the segment.
	step step
	// name is the name of a field or a variant.
	name string
	// index is the index of an item.
	index int
	// key is the key of an entry.
	key any
}

// Field returns the segment of the field name of a record. At the front of
// a path it names the input itself, such as a Go type or a vector. It
// allocates nothing.
func Field(name string) Segment { return Segment{step: field, name: name} }

// Index returns the segment of the item at index i of a sequence. It
// allocates nothing.
func Index(i int) Segment { return Segment{step: index, index: i} }

// Key returns the segment of the entry at key k of a map. It allocates
// nothing.
func Key(k any) Segment { return Segment{step: key, key: k} }

// Element returns the segment of every element of a sequence type, where a
// path names a part of a type rather than of a value. It allocates nothing.
func Element() Segment { return Segment{step: element} }

// Variant returns the segment of the variant name of an enum. It allocates
// nothing.
func Variant(name string) Segment { return Segment{step: variant, name: name} }

// Path is where in an input a fault is: its segments, outermost first.
type Path []Segment

// scratch is the length of the buffer on the stack that the text of an
// index or a key is formatted into.
const scratch = 64

// String returns p as a selector. A field follows a dot, except at the
// front. An index or a key is in brackets, a key of text quoted. An element
// is a pair of empty brackets, and a variant is in parentheses after a dot:
// order.Lines[].Note, counts["a"] and payment.(paid).
//
// # Allocation contract
//
// String allocates the text it returns once, and nothing for the empty
// path. A key whose text is longer than 64 bytes allocates as it is
// formatted, twice.
func (p Path) String() string {
	var b strings.Builder
	b.Grow(p.size())
	var buf [scratch]byte
	for i, s := range p {
		switch s.step {
		case field:
			if i > 0 {
				b.WriteByte('.')
			}
			b.WriteString(s.name)
		case index, key:
			b.WriteByte('[')
			b.Write(s.appendText(buf[:0]))
			b.WriteByte(']')
		case element:
			b.WriteString("[]")
		case variant:
			b.WriteString(".(")
			b.WriteString(s.name)
			b.WriteByte(')')
		}
	}
	return b.String()
}

// size returns the length of the text that String returns for p.
func (p Path) size() int {
	n := 0
	var buf [scratch]byte
	for i, s := range p {
		switch s.step {
		case field:
			n += len(s.name)
			if i > 0 {
				n++
			}
		case index, key:
			n += len("[]") + len(s.appendText(buf[:0]))
		case element:
			n += len("[]")
		case variant:
			n += len(".()") + len(s.name)
		}
	}
	return n
}

// appendText appends to dst the text of the index or the key of s: an index
// in decimal, a key of text quoted, and any other key as fmt prints it.
func (s Segment) appendText(dst []byte) []byte {
	if s.step == index {
		return strconv.AppendInt(dst, int64(s.index), 10)
	}
	if text, ok := s.key.(string); ok {
		return strconv.AppendQuote(dst, text)
	}
	return fmt.Append(dst, s.key)
}
