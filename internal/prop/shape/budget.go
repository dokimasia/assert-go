// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import "go.dokimi.dev/assert/internal/prop/engine"

// limit is the nodes that one value of a recursive shape may use.
const limit = 100

// refID is the id of a ref's generator, which opens no span of its own.
const refID = "ref"

// budget is the limit on the nodes of each value of one shape file, and the
// owner of their count on a case. The case that decodes a value counts its
// nodes, so the budget keeps nothing that a decode changes.
type budget struct {
	// limit is the nodes that one value may use.
	limit int
}

// exhausted reports whether the value in progress in c has used its
// budget.
func (b *budget) exhausted(c *engine.Case) bool {
	return c.Count(b) >= b.limit
}

// stop returns the function that a collection that exits asks before each
// element, which reports whether the value in progress in c has used its
// budget, and nil for a collection that does not exit.
func (b *budget) stop(c *engine.Case, exits bool) func() bool {
	if !exits {
		return nil
	}
	return func() bool { return b.exhausted(c) }
}

// rootOf returns the generator of a recursive shape file's root: of's
// value, whose nodes the case counts from zero while it decodes. It runs
// backwards through of.
func rootOf(of engine.Generator[any], b *budget) engine.Generator[any] {
	decode := func(c *engine.Case) any {
		defer c.Enter(b)()
		return of.Decode(c)
	}
	return engine.NewInvertible(of.ID(), decode, of.Inverse)
}

// refOf returns the generator of the definition name of built, which counts
// one node of the value in progress when the definition is cyclic. It runs
// backwards through the definition.
func refOf(name string, built map[string]engine.Generator[any], cyclic bool, b *budget) engine.Generator[any] {
	decode := func(c *engine.Case) any {
		if cyclic {
			c.Add(b)
		}
		return built[name].Decode(c)
	}
	return engine.NewInvertible(refID, decode, func(v any) ([]engine.Step, any, error) {
		return built[name].Inverse(v)
	})
}
