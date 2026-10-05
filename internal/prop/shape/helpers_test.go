// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape_test

import (
	"math"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/literal"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/shape"
)

// drawn is the label of the one value a test body draws.
const drawn = "value"

// Shapes that the tests of more than one file decode.
const (
	// uint8Shape is an unsigned byte.
	uint8Shape = `{"shape":"int","width":8,"signed":false}`
	// recordShape is a record of an int and a bool.
	recordShape = `{"shape":"record","fields":[["id",` + uint8Shape + `],["ok",{"shape":"bool"}]]}`
)

// allOnes is the integer choice of the largest uint64, whose bits are all
// ones.
var allOnes = choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(math.MaxUint64)}

// read returns the generator of the shape text, failing the test when it
// does not read.
func read(tb testing.TB, text string) engine.Generator[any] {
	tb.Helper()
	g, err := shape.Read([]byte(text))
	assert.NoError(tb, err, "the shape reads")
	return g
}

// decode returns the value that the shape text decodes from a case
// replaying choices, with the run of the case.
func decode(tb testing.TB, text string, choices ...choice.Choice) (any, engine.Execution) {
	tb.Helper()
	g := read(tb, text)
	var got any
	e := engine.Replay(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, choices, nil)
	return got, e
}

// n returns the integer choice of v.
func n(v int64) choice.Choice {
	return choice.Choice{Kind: choice.Integer, Integer: choice.IntOf(v)}
}

// seq returns the sequence choice of the elements.
func seq(elements ...uint32) choice.Choice {
	return choice.Choice{Kind: choice.Sequence, Sequence: elements}
}

// record returns the record of two named fields.
func record(first string, a any, second string, b any) literal.Record {
	return literal.Record{Fields: []literal.Field{{Name: first, Value: a}, {Name: second, Value: b}}}
}

// isFault checks that err is a fault of kind at path, whose reason is
// reason.
func isFault(tb testing.TB, err, kind error, path fault.Path, reason string) {
	tb.Helper()
	assert.ErrorIs(tb, err, kind, "the kind of the fault")
	f := assert.ErrorAs[*fault.Error](tb, err, "a fault")
	assert.Equal(tb, f.Path, path, "where the fault is")
	assert.Equal(tb, f.Reason, reason, "what is wrong")
}

// roundTrips checks that each value that the shape text generates in the
// first 60 cases of seed 9 runs back, as runsBack checks.
func roundTrips(tb testing.TB, text string) {
	tb.Helper()
	g := read(tb, text)
	for index := range uint64(60) {
		var v any
		engine.Generate(func(c *engine.Case) { v = engine.Draw(c, g, drawn) }, 9, index, nil)
		runsBack(tb, text, g, v)
	}
}

// runsBack checks that v, a value of g, the generator of the shape text, runs
// back to choices that decode to the value, and that the decoded value runs
// back to the same choices. A set decodes its elements in the order of their
// choices, so the decoded value of a text that states a set is compared by
// its choices alone.
func runsBack(tb testing.TB, text string, g engine.Generator[any], v any) {
	tb.Helper()
	choices, err := engine.Invert(g, v)
	assert.NoError(tb, err, text+" runs its value back")
	var got any
	engine.Replay(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, choices, nil)
	again, err := engine.Invert(g, got)
	assert.NoError(tb, err, text+" runs the decoded value back")
	assert.True(tb, slices.EqualFunc(again, choices, choice.Choice.Equal),
		text+" runs the decoded value back to the same choices")
	if !strings.Contains(text, `"set"`) {
		assert.Equal(tb, literal.Canonical(got), literal.Canonical(v), text+" decodes the value back")
	}
}
