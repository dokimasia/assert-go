// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

import (
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/internal/prop/tree"
)

// executor runs the cases of the generation phase into the case tree, one
// for each call of its methods, and returns how each case ends there.
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
	return inOrder{body: body, s: s, tree: t}
}

// inOrder is the executor of one worker. Each case walks the case tree as
// it runs, so a repeated case stops at its repeat, before the body goes on
// to exercise the subject.
type inOrder struct {
	// body is the property's body.
	body Body
	// s are the settings of the run.
	s Settings
	// tree is the case tree.
	tree *tree.Tree
}

// replay runs the case that replays choices into the tree.
func (o inOrder) replay(choices []choice.Choice) Execution {
	return execute(o.body, replaying{choices: choices}, o.s.MaxChoices, o.tree, o.s.Clock)
}

// random runs random case index into the tree.
func (o inOrder) random(index uint64) (Execution, []choice.Choice, random.Source) {
	source := random.ForCase(o.s.Seed, index)
	e := execute(o.body, newGenerating(&source), o.s.MaxChoices, o.tree, o.s.Clock)
	return e, e.Case.Choices(), source
}

// edge runs the edge case of the boundary at into the tree.
func (o inOrder) edge(at boundary) Execution {
	return execute(o.body, edge{at: at}, o.s.MaxChoices, o.tree, o.s.Clock)
}

// close returns at once: each case of one worker ended before the next
// started.
func (inOrder) close() {}
