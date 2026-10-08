// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// The parameters of the measured run of steps.
const (
	// queueSteps is the number of sequential steps of the measured case.
	queueSteps = 100
	// queueCaseAllocs are the allocations of the measured case, a whole
	// replayed case of 100 sequential steps of a queue with its spec, which
	// checks the history 101 times, each check continuing the search of the
	// one before it on a goroutine of its own, measured: about 15 for each
	// check.
	queueCaseAllocs = 1538
)

// digit is the generator of the inputs that the tests' actions draw.
var digit = prop.Integer(0, 9)

// TestSteps checks the six parts of a machine's run of steps: the swarm, the
// setup, the sequential steps, the concurrent section, the drain and the
// settling.
func TestSteps(t *testing.T) {
	t.Parallel()

	t.Run("Steps", func(t *testing.T) {
		t.Parallel()

		t.Run("keeps each action by a choice, in order, and takes steps of the kept ones alone", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put, get)) }, 0, 1, 1, 0, 0)
			assert.Equal(t, stepsOf(e), []engine.MachineStep{sequential(get)}, "the step of get, which the case kept")
		})

		t.Run("keeps the last action while no earlier one is kept, whatever its choice states", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put, get)) }, 0, 0, 1, 0, 0)
			assert.Equal(t, choicesOf(e), []uint64{0, 1, 1, 0, 0}, "the swarm choice of get recorded as 1")
			assert.Equal(t, stepsOf(e), []engine.MachineStep{sequential(get)}, "the step of get")
		})

		t.Run("makes no choice for a machine without actions", func(t *testing.T) {
			t.Parallel()
			e := replayed(func(c *prop.Case) { stateful.Steps(c, stateful.Machine[int]{}) }, 1, 1)
			assert.Empty(t, choicesOf(e), "no choice")
		})

		t.Run("checks the history and runs the invariant after setup, each step and the settle", func(t *testing.T) {
			t.Parallel()
			var states []int
			count := 0
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{
					Spec:      counter,
					Actions:   []stateful.Action[int]{incrementOf(&count, 0)},
					Invariant: func(_ *prop.Case, state int) { states = append(states, state) },
				})
			}, 1, 1, 0, 1, 0, 0)
			assert.Equal(t, e.Status, engine.CasePassed, "every check passes")
			assert.Equal(t, states, []int{0, 1, 2, 2}, "the initial state, a state after each step, and the last")
		})

		t.Run("takes a step of the listed action at its index for each flag that continues", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put, get)) }, 1, 1, 1, 1, 1, 0, 0)
			assert.Equal(t, stepsOf(e), []engine.MachineStep{sequential(get), sequential(put)}, "get, then put")
			assert.Equal(t, log.all(), []string{"get/0", "put/0"}, "both on client 0")
		})

		t.Run("requests a step's input from the case inside the step's span, before its run", func(t *testing.T) {
			t.Parallel()
			var got []any
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{{
					Name:  put,
					Input: func(c *prop.Case, _ int) any { return c.Draw(digit, "v") },
					Run:   func(_ *prop.Case, _ int, v any) { got = append(got, v) },
				}}})
			}, 1, 1, 0, 7, 0)
			assert.Equal(t, got, []any{7}, "the drawn input")
			spans := e.Case.Spans()
			assert.Equal(t, []any{spans[0].Label, spans[0].Start, spans[0].End}, []any{put, 1, 4},
				"the step from its flag past its input")
			assert.Equal(t, e.Case.Steps(), []engine.RecordedStep{{MachineStep: sequential(put), Draws: 0}},
				"the step before its draw")
		})

		t.Run("lists the actions enabled in every state of the last check, asking each state once", func(t *testing.T) {
			t.Parallel()
			var seen []int
			low := stateful.Action[int]{
				Name:    "low",
				Enabled: func(state int) bool { seen = append(seen, state); return state < 5 },
				Run:     func(*prop.Case, int, any) {},
			}
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Spec: forked, Actions: []stateful.Action[int]{
					incrementOf(new(int), 0), low,
				}})
			}, 1, 1, 1, 0, 1, 1, 0)
			assert.Equal(t, stepsOf(e), []engine.MachineStep{sequential("increment"), sequential("increment")},
				"low is unlisted once one state of the check is 10")
			assert.Equal(t, slices.Sorted(slices.Values(seen)), []int{0, 1, 2, 10, 11}, "each state once per list")
		})

		t.Run("ends the sequential steps without a flag once no kept action is enabled", func(t *testing.T) {
			t.Parallel()
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{{
					Name: put, Enabled: func(int) bool { return false }, Run: func(*prop.Case, int, any) {},
				}}})
			}, 1, 1, 0)
			assert.Equal(t, choicesOf(e), []uint64{1}, "the swarm choice alone")
		})

		t.Run("lists a section's steps before any runs, then runs client 0's and then the others'", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put), tasks(c)...) },
				1, 0, 1, 2, 0, 1, 0, 0, 0)
			assert.Equal(t, stepsOf(e), []engine.MachineStep{{Action: put, Client: 2}, {Action: put, Client: 0}},
				"the steps in the order listed")
			assert.Equal(t, log.all(), []string{"put/0", "put/2"}, "client 0's step first")
		})

		t.Run("lists the kept actions without Enabled in a section, each with its input", func(t *testing.T) {
			t.Parallel()
			var got []any
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{{
					Name: put, Enabled: func(int) bool { return true }, Run: func(*prop.Case, int, any) {},
				}, {
					Name:  get,
					Input: func(c *prop.Case, _ int) any { return c.Draw(digit, "v") },
					Run:   func(_ *prop.Case, _ int, v any) { got = append(got, v) },
				}}}, tasks(c)...)
			}, 1, 1, 0, 1, 1, 0, 4, 1, 2, 0, 5, 0)
			assert.Equal(t, stepsOf(e), []engine.MachineStep{{Action: get, Client: 1}, {Action: get, Client: 2}},
				"two steps of get, the one action without Enabled")
			assert.Equal(t, got, []any{4, 5}, "the inputs drawn as the section listed its steps")
		})

		t.Run("makes no choice in a section without an action to list", func(t *testing.T) {
			t.Parallel()
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{{
					Name: put, Enabled: func(int) bool { return true }, Run: func(*prop.Case, int, any) {},
				}}}, stateful.Clients(2))
			}, 1, 0, 1, 1)
			assert.Equal(t, choicesOf(e), []uint64{1, 0}, "the swarm choice and the sequential flag")
		})

		t.Run("runs a section on threads once when every step is on client 0", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put), stateful.Clients(2)) },
				1, 0, 1, 0, 0, 0)
			assert.Equal(t, log.all(), []string{"put/0"}, "one run of the case")
		})

		t.Run("takes drain steps of the drain actions, kept or not, until none is enabled", func(t *testing.T) {
			t.Parallel()
			pending := 0
			e := replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{{
					Name: put, Run: func(*prop.Case, int, any) { pending++ },
				}, {
					Name: flush, Drain: true,
					Enabled: func(int) bool { return pending > 0 },
					Run:     func(*prop.Case, int, any) { pending-- },
				}}})
			}, 1, 0, 1, 0, 1, 0, 0, 0, 0)
			drained := engine.MachineStep{Action: flush, Client: -1, Drain: true}
			assert.Equal(t, stepsOf(e), []engine.MachineStep{sequential(put), sequential(put), drained, drained},
				"two puts, and a flush for each")
			assert.Equal(t, choicesOf(e), []uint64{1, 0, 1, 0, 1, 0, 0, 0, 0}, "an index without a flag per flush")
		})

		t.Run("stops the drain at Max steps", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			m := stateful.Machine[int]{Actions: []stateful.Action[int]{logged(flush, log)}}
			m.Actions[0].Drain = true
			e := replayed(func(c *prop.Case) { stateful.Steps(c, m, stateful.Max(2)) }, 1, 0)
			assert.Length(t, stepsOf(e), 2, "two drain steps")
		})

		t.Run("runs Settle on the first state of the last check, then the invariant", func(t *testing.T) {
			t.Parallel()
			var calls []any
			count := 0
			replayed(func(c *prop.Case) {
				stateful.Steps(c, stateful.Machine[int]{
					Spec:      counter,
					Actions:   []stateful.Action[int]{incrementOf(&count, 0)},
					Invariant: func(_ *prop.Case, state int) { calls = append(calls, "invariant", state) },
					Settle:    func(_ *prop.Case, state int) { calls = append(calls, "settle", state) },
				})
			}, 1, 1, 0, 0)
			assert.Equal(t, calls, []any{"invariant", 0, "invariant", 1, "settle", 1, "invariant", 1},
				"the invariant after setup and the step, then the settle and the invariant")
		})

		input := engine.Entry{Label: "v", Value: 5}
		tests := []struct {
			name    string
			give    func(c *prop.Case, calls int)
			entries []engine.Entry
			want    engine.Place
			drawing bool
		}{
			{
				name: "places a swarm choice at the position of its action",
				give: func(c *prop.Case, calls int) {
					names := []string{put}
					if calls > 1 {
						names = append(names, get)
					}
					stateful.Steps(c, machineOf(&runLog{}, names...))
				},
				entries: []engine.Entry{step(put)},
				want:    engine.Place{Part: engine.SwarmPart, Position: 0, Positioned: true, Action: put, Acting: true},
			},
			{
				name: "places the invariant before the first step in setup",
				give: func(c *prop.Case, calls int) {
					m := machineOf(&runLog{}, put)
					m.Invariant = func(c *prop.Case, _ int) { c.Observe(uint64(calls)) }
					stateful.Steps(c, m)
				},
				entries: []engine.Entry{step(put)},
				want:    engine.Place{Part: engine.SetupPart},
			},
			{
				name: "places the index of a sequential step at its position, before the step has its action",
				give: func(c *prop.Case, calls int) {
					m := machineOf(&runLog{}, put, get)
					m.Actions[1].Enabled = func(int) bool { return calls == 1 }
					stateful.Steps(c, m, stateful.Swarm(false))
				},
				entries: []engine.Entry{step(get)},
				want:    engine.Place{Part: engine.SequentialPart, Position: 0, Positioned: true},
			},
			{
				name: "places the input of a sequential step at its position, with its action",
				give: func(c *prop.Case, calls int) {
					stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{varying(put, calls, 1)}})
				},
				entries: []engine.Entry{step(put), input, step(put), input},
				want: engine.Place{
					Part: engine.SequentialPart, Position: 1, Positioned: true, Action: put, Acting: true,
				},
				drawing: true,
			},
			{
				name: "places the input of a step that a concurrent section lists at its position, with its action",
				give: func(c *prop.Case, calls int) {
					stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{varying(put, calls, 0)}},
						tasks(c)...)
				},
				entries: []engine.Entry{concurrent(put, 1), input},
				want: engine.Place{
					Part: engine.ConcurrentPart, Position: 0, Positioned: true, Action: put, Acting: true,
				},
				drawing: true,
			},
			{
				name: "places the run of a concurrent section's steps in the section, without a position",
				give: func(c *prop.Case, calls int) {
					observing := stateful.Action[int]{
						Name: put,
						Run:  func(c *prop.Case, _ int, _ any) { c.Observe(uint64(calls)) },
					}
					stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{observing}}, tasks(c)...)
				},
				entries: []engine.Entry{concurrent(put, 1)},
				want:    engine.Place{Part: engine.ConcurrentPart},
			},
			{
				name: "places the input of a drain step at its position, with its action",
				give: func(c *prop.Case, calls int) {
					done := false
					a := varying(flush, calls, 0)
					a.Drain, a.Enabled = true, func(int) bool { return !done }
					a.Run = func(*prop.Case, int, any) { done = true }
					stateful.Steps(c, stateful.Machine[int]{Actions: []stateful.Action[int]{a}})
				},
				entries: []engine.Entry{drained(flush), input},
				want: engine.Place{
					Part: engine.DrainPart, Position: 0, Positioned: true, Action: flush, Acting: true,
				},
				drawing: true,
			},
			{
				name: "places Settle in settle",
				give: func(c *prop.Case, calls int) {
					m := machineOf(&runLog{}, put)
					m.Settle = func(c *prop.Case, _ int) { c.Observe(uint64(calls)) }
					stateful.Steps(c, m)
				},
				entries: []engine.Entry{step(put)},
				want:    engine.Place{Part: engine.SettlePart},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				want := engine.Where{Place: tt.want, Placed: true}
				if tt.drawing {
					want.Label, want.Drawing = "v", true
				}
				assert.Equal(t, divergedWhere(t, tt.give, tt.entries), want,
					"where the replay of the failing case differs")
			})
		}

		t.Run("leaves the case outside a machine's steps once they end", func(t *testing.T) {
			t.Parallel()
			give := func(c *prop.Case, calls int) {
				stateful.Steps(c, machineOf(&runLog{}, put))
				c.Observe(uint64(calls))
			}
			assert.Equal(t, divergedWhere(t, give, []engine.Entry{step(put)}), engine.Where{},
				"no place after the steps")
		})
	})
}

// divergedWhere runs give in a case that follows entries and then fails,
// and returns where the replay of the failing case differs. give takes the
// number of its call, so the replay, its second call, can differ from the
// case.
func divergedWhere(t *testing.T, give func(c *prop.Case, calls int), entries []engine.Entry) engine.Where {
	t.Helper()
	calls := 0
	r := engine.Run(bodyOf(func(c *prop.Case) {
		calls++
		give(c, calls)
		c.Fatalf("always")
	}), engine.Settings{Cases: 1, MaxChoices: engine.MaxChoices, Shrink: engine.DefaultShrink, Draws: entries})
	assert.Equal(t, r.Outcome, engine.Flaky, "a flaky run")
	return r.Divergence.Where
}

// varying returns the action named name whose input is a draw under "v":
// from 0 to 99 in the step at index at of a later call of a body, and from
// the digits in every other step.
func varying(name string, calls, at int) stateful.Action[int] {
	steps := 0
	return stateful.Action[int]{
		Name: name,
		Input: func(c *prop.Case, _ int) any {
			steps++
			if calls > 1 && steps == at+1 {
				return c.Draw(prop.Integer(0, 99), "v")
			}
			return c.Draw(digit, "v")
		},
		Run: func(*prop.Case, int, any) {},
	}
}

// forked is the spec of a counter whose first increment may have taken it
// to 1 or to 10, and whose later increments add 1. It accepts every output.
var forked = history.Spec[int]{
	Initial: func() int { return 0 },
	Next: func(state int, _ history.Operation) []int {
		if state == 0 {
			return []int{1, 10}
		}
		return []int{state + 1}
	},
}

// fifo is the spec of a queue: put appends its argument, and get returns
// the oldest value, or nil for an empty queue.
var fifo = history.Spec[[]int]{
	Initial: func() []int { return []int{} },
	Next: func(state []int, op history.Operation) [][]int {
		if op.Name == put {
			return [][]int{append(slices.Clip(state), op.Args[0].(int))}
		}
		if len(state) == 0 && op.Output == nil {
			return [][]int{state}
		}
		if len(state) == 0 || op.Output != any(state[0]) {
			return nil
		}
		return [][]int{state[1:]}
	},
}

// queueCase is the body of the measured case: sequential steps of put and
// get over a queue, with the spec of a queue.
var queueCase = bodyOf(func(c *prop.Case) {
	var q []int
	written := 0
	stateful.Steps(c, stateful.Machine[[]int]{
		Spec: fifo,
		Actions: []stateful.Action[[]int]{{
			Name:  put,
			Input: func(*prop.Case, []int) any { written++; return written },
			Run: func(c *prop.Case, client int, v any) {
				call := c.History().Invoke(client, put, []any{v})
				q = append(q, v.(int))
				call.OK(nil)
			},
		}, {
			Name: get,
			Run: func(c *prop.Case, client int, _ any) {
				call := c.History().Invoke(client, get, nil)
				if len(q) == 0 {
					call.OK(nil)
					return
				}
				call.OK(q[0])
				q = q[1:]
			},
		}},
	}, stateful.Swarm(false), stateful.Max(queueSteps))
})

// queueChoices are the choices of the measured case: a flag and an index
// for each step, put and get in turn.
var queueChoices = func() []choice.Choice {
	values := make([]uint64, 0, 2*queueSteps)
	for i := range queueSteps {
		values = append(values, 1, uint64(i%2))
	}
	return integers(values)
}()

// TestStepsAllocs checks the ceiling of a case of steps.
func TestStepsAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { engine.Replay(queueCase, queueChoices, nil) }, queueCaseAllocs,
		"a case of 100 steps of a queue allocates its checks")
}

// BenchmarkSteps measures a case of 100 sequential steps of a queue with
// its spec, which checks the history 101 times.
func BenchmarkSteps(b *testing.B) {
	b.Run("Steps", func(b *testing.B) {
		var got engine.Execution
		c := bench.Start(b).MaxAllocs(queueCaseAllocs)
		defer c.End()
		for c.Loop() {
			got = engine.Replay(queueCase, queueChoices, nil)
		}
		assert.Length(b, got.Case.Steps(), queueSteps, "every step")
	})
}
