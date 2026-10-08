// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pattern_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/pattern"
)

// parseAllocs are the allocations of Parse on the pattern of a hexadecimal
// identifier, measured: the parser and its characters, and the parsed
// pieces and classes.
const parseAllocs = 16

// identifier is the pattern of the benchmark and the allocation check.
const identifier = `[a-f0-9]{4}-\d{2}`

// TestPatternAllocs checks the allocation ceiling of Parse.
func TestPatternAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _, _ = pattern.Parse(identifier) }, parseAllocs,
		"Parse allocates the pattern's characters and its pieces")
}

// BenchmarkPattern measures Parse.
func BenchmarkPattern(b *testing.B) {
	b.Run("Parse", func(b *testing.B) {
		var got pattern.Node
		c := bench.Start(b).MaxAllocs(parseAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = pattern.Parse(identifier)
		}
		_, sequence := got.(pattern.Sequence)
		assert.True(b, sequence, "the identifier parses to a sequence")
	})
}
