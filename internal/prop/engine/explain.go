// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Relevance -linecomment -output=explain.string_gen.go

// explainFillings is the number of random fillings the explain phase tries
// for each draw.
const explainFillings = 4

// Relevance is what the explain phase found for one draw.
type Relevance uint8

const (
	// Untested is a draw that made no choice, none of whose fillings
	// decoded a value, or whose fillings the budget did not reach.
	Untested Relevance = 0 // untested
	// AnyValueFails is a draw for which every filling fails the same way.
	AnyValueFails Relevance = 1 // any-value-fails
	// ValueMatters is a draw for which a filling passes or fails another
	// way.
	ValueMatters Relevance = 2 // value-matters
)

// Valid reports whether r is one of the three relevances.
func (r Relevance) Valid() bool {
	return r <= ValueMatters
}

// Explained is one draw of the counterexample, explained.
type Explained struct {
	// Label is the label of the draw.
	Label string
	// Value is the drawn value.
	Value any
	// Relevance is what the explain phase found for the draw.
	Relevance Relevance
	// NearestPassing is the value one step towards the target, for an
	// integer or a duration that matters, where that value passes. It is
	// nil otherwise.
	NearestPassing any
}

// explain explains each draw of a failure's case, spending the runs left
// in the budget.
//
// It fills each draw that made a choice with four random values, the n-th
// from the stream of seed + (draw + 1) * 2^32 + n. A draw for which every
// filling that decoded a value still fails the same way is one where any
// value fails. For an integer or a duration that matters, one step towards
// its target that passes is its nearest passing value.
func explain(sh *shrinker, f *failure, seed uint64) []Explained {
	draws, spans := f.execution.Case.Draws(), f.execution.Case.Spans()
	out := make([]Explained, len(draws))
	for number, drawn := range draws {
		out[number] = Explained{Label: drawn.Label, Value: drawn.Value}
		span := spans[drawn.Span]
		if span.Start == span.End {
			continue
		}
		relevance := fill(sh, f, span, drawn.source, seed+(uint64(number)+1)<<32)
		if relevance == ValueMatters {
			out[number].NearestPassing, relevance = nearest(sh, f.nodes, span, drawn.source)
		}
		out[number].Relevance = relevance
	}
	return out
}

// fill runs the fillings of one draw, the n-th from the stream of base + n,
// up to Workers of them at once, and takes their runs in order, each
// charged to the budget. A filling whose decode returns no value is not a
// value of the draw, and is skipped at no cost. It returns [ValueMatters]
// at the first filling that passes or fails another way, [AnyValueFails]
// when every filling that decoded fails the same way, and [Untested] when
// none decoded or the budget or the time runs out first.
func fill(sh *shrinker, f *failure, span Span, g erased, base uint64) Relevance {
	relevance := Untested
	for next := uint64(0); ; {
		batch := make([][]choice.Choice, 0, sh.workers)
		for ; next < explainFillings && len(batch) < sh.workers; next++ {
			source := random.New(base + next)
			if choices, ok := filled(sh, f.nodes, span, g, &source); ok {
				batch = append(batch, choices)
			}
		}
		if len(batch) == 0 {
			return relevance
		}
		count := min(len(batch), sh.room())
		if count == 0 {
			return Untested
		}
		for _, e := range sh.runAll(batch[:count]) {
			sh.runs++
			if e.Status != CaseFailed || e.Identity != f.identity {
				return ValueMatters
			}
		}
		if count < len(batch) {
			return Untested
		}
		relevance = AnyValueFails
	}
}

// filled returns the choices of nodes with one draw's span replaced by the
// choices that a fresh decode of g, capped at [MaxChoices], draws from
// source. It reports false when the decode returns no value: it rejects,
// passes the cap or panics.
func filled(sh *shrinker, nodes []node, span Span, g erased, source *random.Source) ([]choice.Choice, bool) {
	fresh := execute(func(c *Case) { g.decode(c) }, newGenerating(source), MaxChoices, nil, sh.s.Clock)
	if fresh.Status != CasePassed {
		return nil, false
	}
	choices := choicesOf(nodes)
	return slices.Concat(choices[:span.Start], fresh.Case.Choices(), choices[span.End:]), true
}

// nearest returns the value one step towards the target of an integer or a
// duration draw, where that value passes, and the relevance that the
// budget left: [ValueMatters], or [Untested] when the budget ran out.
func nearest(sh *shrinker, nodes []node, span Span, g erased) (any, Relevance) {
	if !g.integer {
		return nil, ValueMatters
	}
	value, target := nodes[span.Start].c.Integer, g.bounds.Target()
	if value == target {
		return nil, ValueMatters
	}
	stepped := towards(value, target, 1)
	e, ok := sh.run(choicesOf(replaced(nodes, span.Start, integerChoice(stepped))))
	if !ok {
		return nil, Untested
	}
	if e.Status != CasePassed {
		return nil, ValueMatters
	}
	return g.value(stepped), ValueMatters
}
