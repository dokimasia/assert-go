// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matching_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/matching"
)

// drawn is the label of the one string a test body draws.
const drawn = "s"

// generatorOf returns the generator of text, failing the test when text is
// outside the portable subset.
func generatorOf(tb testing.TB, text string) engine.Generator[string] {
	tb.Helper()
	g, err := matching.StringMatching(text)
	assert.NoError(tb, err, "the pattern is in the portable subset")
	return g
}

// decode returns the string that text decodes from a case replaying the
// integer choices of values, with the run of that case.
func decode(tb testing.TB, text string, values ...uint64) (string, engine.Execution) {
	tb.Helper()
	g := generatorOf(tb, text)
	choices := make([]choice.Choice, len(values))
	for i, v := range values {
		choices[i] = choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)}
	}
	var got string
	e := engine.Replay(func(c *engine.Case) { got = engine.Draw(c, g, drawn) }, choices, nil)
	return got, e
}
