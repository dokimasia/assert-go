// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"fmt"
	"slices"
	"sync"

	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// The names of the actions of the tests' machines.
const (
	put   = "put"
	get   = "get"
	flush = "flush"
)

// seed is the seed of the random cases of the tests.
const seed = 7

// runLog is the log of the runs of a machine's actions, safe for the
// clients of a concurrent section.
type runLog struct {
	// mu guards runs.
	mu sync.Mutex
	// runs are the runs in order, each the action's name and its client.
	runs []string
}

// add logs the run of action on client.
func (l *runLog) add(action string, client int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.runs = append(l.runs, fmt.Sprintf("%s/%d", action, client))
}

// all returns a copy of the runs, in order.
func (l *runLog) all() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.runs)
}

// logged returns an action named name without a model's state, whose run
// logs itself in log.
func logged(name string, log *runLog) stateful.Action[int] {
	return stateful.Action[int]{Name: name, Run: func(_ *prop.Case, client int, _ any) { log.add(name, client) }}
}

// machineOf returns a machine without a model of the actions named names,
// each logged in log.
func machineOf(log *runLog, names ...string) stateful.Machine[int] {
	m := stateful.Machine[int]{}
	for _, name := range names {
		m.Actions = append(m.Actions, logged(name, log))
	}
	return m
}

// tasks returns the options of a concurrent section on the clients 0 to 2,
// which run as tasks of a uniform scheduler of c.
func tasks(c *prop.Case) []stateful.Option {
	return []stateful.Option{stateful.Clients(2), stateful.Tasks(stateful.NewScheduler(c, stateful.Uniform()))}
}

// integers returns the integer choices of values.
func integers(values []uint64) []choice.Choice {
	out := make([]choice.Choice, len(values))
	for i, v := range values {
		out[i] = choice.Choice{Kind: choice.Integer, Integer: choice.UintOf(v)}
	}
	return out
}

// bodyOf returns body as a body of the engine.
func bodyOf(body func(c *prop.Case)) engine.Body {
	return func(c *engine.Case) { body((*prop.Case)(c)) }
}

// replayed runs body on a case that replays the integer choices, outside a
// run, and returns how the case ended.
func replayed(body func(c *prop.Case), choices ...uint64) engine.Execution {
	return engine.Replay(bodyOf(body), integers(choices), nil)
}

// choicesOf returns the integer choices that the case of e recorded.
func choicesOf(e engine.Execution) []uint64 {
	choices := e.Case.Choices()
	out := make([]uint64, len(choices))
	for i, c := range choices {
		out[i] = c.Integer.Magnitude()
	}
	return out
}

// stepsOf returns the steps that the case of e recorded, without the draws
// before each.
func stepsOf(e engine.Execution) []engine.MachineStep {
	recorded := e.Case.Steps()
	out := make([]engine.MachineStep, len(recorded))
	for i, s := range recorded {
		out[i] = s.MachineStep
	}
	return out
}

// sequential returns the step of action outside a concurrent section.
func sequential(action string) engine.MachineStep {
	return engine.MachineStep{Action: action, Client: -1}
}

// step returns the entry of a sequential step of action.
func step(action string) engine.Entry {
	return engine.Entry{Step: &engine.MachineStep{Action: action, Client: -1}}
}

// concurrent returns the entry of a step of action on client.
func concurrent(action string, client int) engine.Entry {
	return engine.Entry{Step: &engine.MachineStep{Action: action, Client: client}}
}

// drained returns the entry of a drain step of action.
func drained(action string) engine.Entry {
	return engine.Entry{Step: &engine.MachineStep{Action: action, Client: -1, Drain: true}}
}

// counter is the model of a counter whose increment returns the new count.
var counter = history.Model[int]{
	Init: func() int { return 0 },
	Step: func(state int, op history.Op) []int {
		if op.Known && op.Output != any(state+1) {
			return nil
		}
		return []int{state + 1}
	},
}

// incrementOf returns the action increment of a counter at count, whose
// run records the call in the case's history, and adds skip and one to
// the count.
func incrementOf(count *int, skip int) stateful.Action[int] {
	return stateful.Action[int]{
		Name: "increment",
		Run: func(c *prop.Case, client int, _ any) {
			call := c.History().Invoke(client, "increment", nil)
			*count += 1 + skip
			call.OK(*count)
		},
	}
}
