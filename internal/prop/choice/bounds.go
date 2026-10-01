// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package choice

import "errors"

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
