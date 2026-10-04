// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"go.dokimi.dev/assert/internal/literal"
)

// boundsText is the length of the longest text of bounds: float bounds that
// admit NaN, whose two bounds have 24 characters each.
const boundsText = 80

// numberText is the length of the longest number in the text of bounds: a
// float of 24 characters, such as -2.2250738585072014e-308.
const numberText = 24

// ErrEmpty reports bounds or [Sizes] that admit no value.
var ErrEmpty = errors.New("choice: the bounds admit no value")

// Bounds are the bounds of one choice request, of one of the three kinds.
//
// Bounds are comparable, and == reports whether two requests have the
// same kind and the same bounds. The case tree compares them to find a
// body that requested other choices after the same values. The zero value
// is the integer bounds [0, 0].
type Bounds struct {
	// kind names the field that contains the bounds.
	kind Kind
	// integer, float and sequence are the bounds of each kind, and only
	// the one kind names is set.
	integer  IntegerBounds
	float    FloatBounds
	sequence SequenceBounds
}

// OfInteger returns integer bounds as [Bounds].
func OfInteger(b IntegerBounds) Bounds {
	return Bounds{kind: Integer, integer: b}
}

// OfFloat returns float bounds as [Bounds].
func OfFloat(b FloatBounds) Bounds {
	return Bounds{kind: Float, float: b}
}

// OfSequence returns sequence bounds as [Bounds].
func OfSequence(b SequenceBounds) Bounds {
	return Bounds{kind: Sequence, sequence: b}
}

// Kind returns the kind of choice the bounds request.
func (b Bounds) Kind() Kind {
	return b.kind
}

// Integer returns the integer bounds. They are the zero value when the
// kind is not [Integer].
func (b Bounds) Integer() IntegerBounds {
	return b.integer
}

// Float returns the float bounds. They are the zero value when the kind is
// not [Float].
func (b Bounds) Float() FloatBounds {
	return b.float
}

// Sequence returns the sequence bounds. They are the zero value when the
// kind is not [Sequence].
func (b Bounds) Sequence() SequenceBounds {
	return b.sequence
}

// Target returns the simplest choice the bounds admit.
func (b Bounds) Target() Choice {
	if b.kind == Integer {
		return Choice{Kind: Integer, Integer: b.integer.Target()}
	}
	if b.kind == Float {
		return Choice{Kind: Float, Float: b.float.Target()}
	}
	return Choice{Kind: Sequence, Sequence: b.sequence.Target()}
}

// Admits reports whether c is a choice of the bounds' kind that they
// admit.
func (b Bounds) Admits(c Choice) bool {
	if c.Kind != b.kind {
		return false
	}
	if b.kind == Integer {
		return b.integer.Admits(c.Integer)
	}
	if b.kind == Float {
		return b.float.Admits(c.Float)
	}
	return b.sequence.Admits(c.Sequence)
}

// Coerce fits a recorded choice to the bounds, as each kind's Coerce
// states.
func (b Bounds) Coerce(c Choice) Choice {
	if b.kind == Integer {
		return Choice{Kind: Integer, Integer: b.integer.Coerce(c)}
	}
	if b.kind == Float {
		return Choice{Kind: Float, Float: b.float.Coerce(c)}
	}
	return Choice{Kind: Sequence, Sequence: b.sequence.Coerce(c)}
}

// Key returns the sort key of c under the bounds. The key of a choice the
// bounds do not admit is unspecified.
func (b Bounds) Key(c Choice) Key {
	if b.kind == Integer {
		return b.integer.Key(c.Integer)
	}
	if b.kind == Float {
		return FloatKey(c.Float)
	}
	return SequenceKey(c.Sequence)
}

// integerJSON is integer bounds as the corpus states a request's bounds.
type integerJSON struct {
	Kind string `json:"kind"`
	Min  any    `json:"min"`
	Max  any    `json:"max"`
}

// floatJSON is float bounds as the corpus states a request's bounds.
type floatJSON struct {
	Kind     string `json:"kind"`
	Min      any    `json:"min"`
	Max      any    `json:"max"`
	AllowNaN bool   `json:"allow_nan"`
	Width    uint8  `json:"width"`
}

// sequenceJSON is sequence bounds as the corpus states a request's bounds.
type sequenceJSON struct {
	Kind    string `json:"kind"`
	K       uint32 `json:"k"`
	MinSize int    `json:"min_size"`
	MaxSize *int   `json:"max_size"`
}

// MarshalJSON returns the bounds as the corpus states a request's bounds,
// and as the record of a run states one side of a divergence. The JSON
// object states the kind, and then:
//
//   - For integer bounds, the least and the greatest value.
//   - For float bounds, the least and the greatest value, whether NaN is a
//     value, and the width.
//   - For sequence bounds, the number of element values, and the least and
//     the greatest length. The greatest length is null for a sequence
//     without one.
//
// MarshalJSON writes each value in the JSON form of its typed literal. An
// integer beyond 2^53 - 1 in magnitude becomes a decimal string, and a
// float that JSON has no number for becomes its name.
//
// # Allocation contract
//
// MarshalJSON allocates the JSON and the values that it encodes: 15
// allocations for integer bounds of the whole signed range.
func (b Bounds) MarshalJSON() ([]byte, error) {
	switch b.kind {
	case Integer:
		out := integerJSON{Kind: Integer.String(), Min: plainInt(b.integer.lo), Max: plainInt(b.integer.hi)}
		return json.Marshal(out)
	case Float:
		lo, _ := literal.Plain(b.float.lo)
		hi, _ := literal.Plain(b.float.hi)
		return json.Marshal(floatJSON{
			Kind: Float.String(), Min: lo, Max: hi, AllowNaN: b.float.nan == AdmitNaN, Width: uint8(b.float.width),
		})
	default:
		out := sequenceJSON{Kind: Sequence.String(), K: b.sequence.k, MinSize: b.sequence.sizes.Min()}
		if most, bounded := b.sequence.sizes.Max(); bounded {
			out.MaxSize = &most
		}
		return json.Marshal(out)
	}
}

// plainInt returns the JSON value of the integer literal of i.
func plainInt(i Int) any {
	if v, ok := i.Int64(); ok {
		plain, _ := literal.Plain(v)
		return plain
	}
	plain, _ := literal.Plain(i.magnitude)
	return plain
}

// String returns the bounds as the record of a run states a request:
// "integer in [0, 9]", "float in [0, 1] of width 64", the same with
// " or NaN" when NaN is a value, and "sequence of 0 to 8 values below 256"
// or "sequence of 2 or more values below 256".
//
// A float is in the shortest decimal that names it, as the verb %v of
// package fmt prints it.
//
// # Allocation contract
//
// String allocates the text it returns: one allocation.
func (b Bounds) String() string {
	var s strings.Builder
	s.Grow(boundsText)
	var number [numberText]byte
	switch b.kind {
	case Integer:
		s.WriteString("integer in [")
		s.Write(b.integer.lo.appendText(number[:0]))
		s.WriteString(", ")
		s.Write(b.integer.hi.appendText(number[:0]))
		s.WriteByte(']')
	case Float:
		s.WriteString("float in [")
		s.Write(strconv.AppendFloat(number[:0], b.float.lo, 'g', -1, 64))
		s.WriteString(", ")
		s.Write(strconv.AppendFloat(number[:0], b.float.hi, 'g', -1, 64))
		s.WriteString("] of width ")
		s.Write(strconv.AppendUint(number[:0], uint64(b.float.width), 10))
		if b.float.nan == AdmitNaN {
			s.WriteString(" or NaN")
		}
	default:
		s.WriteString("sequence of ")
		s.Write(strconv.AppendInt(number[:0], int64(b.sequence.sizes.minSize), 10))
		if most, bounded := b.sequence.sizes.Max(); bounded {
			s.WriteString(" to ")
			s.Write(strconv.AppendInt(number[:0], int64(most), 10))
		} else {
			s.WriteString(" or more")
		}
		s.WriteString(" values below ")
		s.Write(strconv.AppendUint(number[:0], uint64(b.sequence.k), 10))
	}
	return s.String()
}
