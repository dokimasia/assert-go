// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"slices"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/record"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Relevance -linecomment -output=explain.string_gen.go

// explainFillings is the number of random fillings the explain phase tries
// for each draw.
const explainFillings = 4

// Relevance is what the explain phase found for one draw.
type Relevance uint8

const (
	// Untested is a draw that made no choice, none of whose fillings
	// decoded a value, or whose fillings did not all run within the budget.
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
// up to Workers of them at once, and takes their runs in order, each with
// the repeats that its case asks for and charged to the budget, as charge
// does. A filling whose decode returns no value is not a value of the draw,
// and is skipped without a charge to the budget. It returns [ValueMatters]
// at the first filling that passes or fails another way, [AnyValueFails]
// when every filling that decoded fails the same way, and [Untested] when
// none decoded or the budget or the time runs out first.
func fill(sh *shrinker, f *failure, span Span, g erased, base uint64) Relevance {
	relevance := Untested
	for next := uint64(0); ; {
		sh.batch = sh.batch[:0]
		for ; next < explainFillings && len(sh.batch) < sh.workers; next++ {
			if choices, ok := filled(sh, f.nodes, span, g, random.New(base+next)); ok {
				sh.batch = append(sh.batch, choices)
			}
		}
		if len(sh.batch) == 0 {
			return relevance
		}
		room := sh.room()
		count := min(len(sh.batch), room)
		if count == 0 {
			return Untested
		}
		runs := sh.runAll(sh.batch[:count])
		taken := 0
		for ; taken < len(runs) && room > 0; taken++ {
			e := sh.charge(runs[taken], room, record.Explain)
			room -= e.runs()
			same := e.Status == CaseFailed && e.Identity == f.identity
			sh.release(e)
			if !same {
				sh.release(runs[taken+1:]...)
				return ValueMatters
			}
		}
		sh.release(runs[taken:]...)
		if taken < len(sh.batch) {
			return Untested
		}
		relevance = AnyValueFails
	}
}

// filled returns the choices of nodes with one draw's span replaced by the
// choices that a fresh decode of g, capped at [MaxChoices], draws from
// source. It reports false when the decode returns no value: it rejects,
// passes the cap or panics. The decode runs on a spare case of sh, with
// sh's generating provider.
func filled(sh *shrinker, nodes []node, span Span, g erased, source random.Source) ([]choice.Choice, bool) {
	sh.generating.reset(source)
	c := sh.spareCase()
	c.recycle(sh.generating, Settings{MaxChoices: MaxChoices, Clock: sh.s.Clock}, nil)
	fresh := finish(c, func(c *Case) { g.decode(c) })
	defer sh.release(fresh)
	if fresh.Status != CasePassed {
		return nil, false
	}
	choices := choicesOf(nodes)
	return slices.Concat(choices[:span.Start], fresh.Case.record(), choices[span.End:]), true
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
	e, ok := sh.run(choicesOf(replaced(nodes, span.Start, integerChoice(stepped))), record.Explain)
	if !ok {
		return nil, Untested
	}
	sh.release(e)
	if e.Status != CasePassed {
		return nil, ValueMatters
	}
	return g.value(stepped), ValueMatters
}
