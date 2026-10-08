// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package tree_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/tree"
)

// step is one choice of a walked case: the bounds of its request and the
// value it took.
type step struct {
	// bounds are the bounds of the request.
	bounds choice.Bounds
	// value is the choice the case took.
	value choice.Choice
}

// walk follows a case of steps down tr and ends it. It returns the first
// error of a step, or the error of the end.
func walk(tr *tree.Tree, steps ...step) error {
	w := tr.Walk()
	for _, s := range steps {
		if err := w.Step(s.bounds, s.value); err != nil {
			return err
		}
	}
	return w.End()
}

// integerBounds returns the bounds of an integer choice in [0, hi],
// failing the test when they are invalid.
func integerBounds(tb testing.TB, hi int64) choice.Bounds {
	tb.Helper()
	b, err := choice.NewIntegerBounds(choice.Int{}, choice.IntOf(hi))
	assert.NoError(tb, err, "the bounds are valid")
	return choice.OfInteger(b)
}

// floatBounds returns the bounds of a binary64 choice in [lo, hi] under
// the NaN policy, failing the test when they are invalid.
func floatBounds(tb testing.TB, lo, hi float64, nan choice.NaNPolicy) choice.Bounds {
	tb.Helper()
	b, err := choice.NewFloatBounds(lo, hi, nan, choice.Width64)
	assert.NoError(tb, err, "the bounds are valid")
	return choice.OfFloat(b)
}

// sequenceBounds returns the bounds of a sequence of integers in [0, k)
// with lengths from minSize to maxSize, failing the test when they are
// invalid.
func sequenceBounds(tb testing.TB, k uint32, minSize, maxSize int) choice.Bounds {
	tb.Helper()
	sizes, err := choice.NewSizes(minSize, maxSize)
	assert.NoError(tb, err, "the sizes are valid")
	b, err := choice.NewSequenceBounds(k, sizes)
	assert.NoError(tb, err, "the bounds are valid")
	return choice.OfSequence(b)
}

// integer returns an integer choice of v.
func integer(v int64) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(v)}
}

// float returns a float choice of v.
func float(v float64) choice.Choice {
	return choice.Choice{Kind: choice.Float, Float: v}
}

// sequence returns a sequence choice of the elements.
func sequence(elements ...uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}
