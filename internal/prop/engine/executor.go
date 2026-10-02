// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/tree"
)

// executor runs the cases of the generation phase into the case tree, one
// for each call of its methods, and returns how each case ends there. The
// execution and the choices that a call returns are valid until the next
// call.
type executor interface {
	// replay runs the case that replays choices and then takes targets:
	// the simplest case for no choices, and a prefix case otherwise.
	replay(choices []choice.Choice) Execution
	// random runs random case index, and returns how it ended, the
	// choices it made, and its source after them.
	random(index uint64) (Execution, []choice.Choice, random.Source)
	// edge runs the edge case of the boundary at.
	edge(at boundary) Execution
	// close ends every case that the run started and no longer needs, and
	// returns once each has ended.
	close()
}

// newExecutor returns the executor of a run of s: on one worker, one that
// runs each case when its method is called, and on more, one that runs
// cases ahead.
func newExecutor(body Body, s Settings, t *tree.Tree) executor {
	if s.Workers > 1 {
		return newAhead(body, s, t)
	}
	return &inOrder{body: body, s: s, walker: t.Walk(), generating: newGenerating(random.Source{})}
}

// inOrder is the executor of one worker. Each case walks the case tree as
// it runs, so a repeated case stops at its repeat, before the body goes on
// to exercise the subject.
//
// A case ends before the next one starts, so the cases share one walker,
// one generating provider and the storage of one case, which each call
// starts over.
type inOrder struct {
	// body is the property's body.
	body Body
	// s are the settings of the run.
	s Settings
	// walker walks each case down the case tree, from the root.
	walker *tree.Walker
	// generating supplies the values of each random case from the case's
	// stream.
	generating *generating
	// last is the case of the last call, whose storage the next call
	// reuses, and nil before the first call.
	last *Case
	// replayed is the copy of the choices of the case that replay runs, which
	// may be the record of the last call's case.
	replayed []choice.Choice
}

// replay runs the case that replays choices into the tree.
func (o *inOrder) replay(choices []choice.Choice) Execution {
	o.replayed = append(o.replayed[:0], choices...)
	return o.execute(replaying{choices: o.replayed})
}

// random runs random case index into the tree. The choices it returns are
// the case's record, which the caller does not change.
func (o *inOrder) random(index uint64) (Execution, []choice.Choice, random.Source) {
	o.generating.reset(random.ForCase(o.s.Seed, index))
	e := o.execute(o.generating)
	return e, e.Case.record(), o.generating.source
}

// edge runs the edge case of the boundary at into the tree.
func (o *inOrder) edge(at boundary) Execution {
	return o.execute(edge{at: at})
}

// execute runs the next case, whose values come from p, from the root of
// the case tree, on the storage of the last call's case.
func (o *inOrder) execute(p provider) Execution {
	o.walker.Restart()
	if o.last == nil {
		o.last = newCase(p, o.s.MaxChoices, o.walker, o.s.Clock)
	} else {
		o.last.recycle(p, o.s.MaxChoices, o.walker, o.s.Clock)
	}
	return finish(o.last, o.body)
}

// close returns at once: each case of one worker ended before the next
// started.
func (*inOrder) close() {}
