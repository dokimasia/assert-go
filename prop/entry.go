// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"strings"

	"go.dokimi.dev/assert/internal/prop/engine"
)

// Entry is one entry of a counterexample, in request order: a [Drawn] value
// or a [Step] of a machine. Drawn and Step are its only types.
type Entry interface {
	// writeLine writes the line of the entry in the sentence of a run's
	// record.
	writeLine(b *strings.Builder)
	// brief returns the entry as another failure's counterexample states
	// it, which explains no draw.
	brief() any
}

// counterexampleOf returns the entries of e's case in request order: its
// draws, each with the explanation at its index, and before each draw the
// steps that the case recorded before it. explained is the run's
// explanation of e, which states one entry for each draw, or nil for a case
// that the run did not explain.
func counterexampleOf(e engine.Execution, explained []engine.Explained) []Entry {
	draws, steps := e.Case.Draws(), e.Case.Steps()
	out := make([]Entry, 0, len(draws)+len(steps))
	for i, d := range draws {
		for len(steps) > 0 && steps[0].Draws <= i {
			out = append(out, Step(steps[0].MachineStep))
			steps = steps[1:]
		}
		drawn := Drawn{Label: d.Label, Value: d.Value}
		if i < len(explained) {
			drawn.Relevance, drawn.NearestPassing = Relevance(explained[i].Relevance), explained[i].NearestPassing
		}
		out = append(out, drawn)
	}
	for _, s := range steps {
		out = append(out, Step(s.MachineStep))
	}
	return out
}
