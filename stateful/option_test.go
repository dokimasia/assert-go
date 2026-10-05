// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/random"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// optionAllocs are the allocations of an option that its call keeps: the
// closure of its setting, measured.
const optionAllocs = 1

// optionCases is the number of random cases whose first flag the tests of
// Mean compare with the random package's coin.
const optionCases = 40

// repeated returns the choices of a case that keeps put, takes no
// sequential step, and lists one step of put on client 1 in a section of
// two clients.
var repeated = []uint64{1, 0, 1, 1, 0, 0}

// TestOption checks that each option of Steps sets its setting, the
// definition's defaults, and the panic of each option below its minimum.
func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("Mean", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []stateful.Option
			want uint64
		}{
			{name: "continues the first step by a coin of 30 in 31 by default", want: 30},
			{
				name: "continues the first step by a coin of the mean in one more",
				give: []stateful.Option{stateful.Mean(5)},
				want: 5,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				opts := append([]stateful.Option{stateful.Swarm(false)}, tt.give...)
				for index := range uint64(optionCases) {
					log := &runLog{}
					e := engine.Generate(bodyOf(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put), opts...) }),
						seed, index, nil)
					twin := random.ForCase(seed, index)
					assert.Equal(t, len(stepsOf(e)) > 0, twin.Coin(tt.want, tt.want+1), "the first flag of the case")
				}
			})
		}

		t.Run("panics for n below 0", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.Mean(-1) }, "a negative mean")
			assert.Equal(t, got, any("stateful: Mean(-1) is below 0"), "the panic names the option")
		})
	})

	t.Run("Max", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []stateful.Option
			want int
		}{
			{name: "stops the sequential steps at 100 by default", want: 100},
			{name: "stops the sequential steps at n", give: []stateful.Option{stateful.Max(2)}, want: 2},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				opts := append([]stateful.Option{stateful.Swarm(false)}, tt.give...)
				log := &runLog{}
				continuing := slices.Repeat([]uint64{1, 0}, 101)
				e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put), opts...) }, continuing...)
				assert.Length(t, stepsOf(e), tt.want, "a step for each flag up to the maximum")
			})
		}

		t.Run("panics for n below 0", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.Max(-1) }, "a negative maximum")
			assert.Equal(t, got, any("stateful: Max(-1) is below 0"), "the panic names the option")
		})
	})

	t.Run("Swarm", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []stateful.Option
			want []engine.MachineStep
		}{
			{
				name: "chooses the actions that a case keeps by default",
				want: []engine.MachineStep{sequential(put)},
			},
			{
				name: "keeps every action without a choice when off",
				give: []stateful.Option{stateful.Swarm(false)},
				want: []engine.MachineStep{sequential(get), sequential(put)},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				log := &runLog{}
				e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put, get), tt.give...) },
					1, 1, 1, 0, 0)
				assert.Equal(t, stepsOf(e), tt.want, "the steps that the choices after the swarm take")
			})
		}
	})

	t.Run("Clients", func(t *testing.T) {
		t.Parallel()

		t.Run("runs no concurrent section for one client by default", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put)) }, repeated...)
			assert.Equal(t, choicesOf(e), []uint64{1, 0}, "the swarm choice and the sequential flag")
		})

		t.Run("runs a section on the clients 0 to n for n of 2 or more", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) {
				s := stateful.NewScheduler(c, stateful.Uniform())
				stateful.Steps(c, machineOf(log, put), stateful.Clients(2), stateful.Tasks(s))
			}, 1, 0, 1, 2, 0, 0)
			assert.Equal(t, stepsOf(e), []engine.MachineStep{{Action: put, Client: 2}}, "the step of client 2")
			assert.Equal(t, log.all(), []string{"put/2"}, "the run on client 2")
		})

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.Clients(0) }, "no client")
			assert.Equal(t, got, any("stateful: Clients(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("Concurrent", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []stateful.Option
			want int
		}{
			{name: "stops a section at 16 steps by default", want: 16},
			{name: "stops a section at n steps", give: []stateful.Option{stateful.Concurrent(1)}, want: 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				log := &runLog{}
				continuing := append([]uint64{1, 0}, slices.Repeat([]uint64{1, 0, 0}, 17)...)
				e := replayed(func(c *prop.Case) {
					stateful.Steps(c, machineOf(log, put), append(tasks(c), tt.give...)...)
				}, continuing...)
				assert.Length(t, stepsOf(e), tt.want, "a step for each flag up to the maximum")
			})
		}

		t.Run("panics for n below 0", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.Concurrent(-1) }, "a negative maximum")
			assert.Equal(t, got, any("stateful: Concurrent(-1) is below 0"), "the panic names the option")
		})
	})

	t.Run("Tasks", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the clients of a section as tasks of s, released by the case's choices", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) {
				s := stateful.NewScheduler(c, stateful.Uniform())
				stateful.Steps(c, machineOf(log, put), stateful.Clients(2), stateful.Tasks(s))
			}, 1, 0, 1, 1, 0, 1, 2, 0, 0, 1, 0)
			assert.Equal(t, log.all(), []string{"put/2", "put/1"}, "the task of client 2 by the release 1")
			assert.Equal(t, choicesOf(e), []uint64{1, 0, 1, 1, 0, 1, 2, 0, 0, 1, 0}, "the two releases last")
		})

		t.Run("panics for a nil scheduler", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.Tasks(nil) }, "no scheduler")
			assert.Equal(t, got, any("stateful: Tasks(nil) states no scheduler"), "the panic names the option")
		})
	})

	t.Run("Repeat", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []stateful.Option
			want []string
		}{
			{
				name: "runs a case whose section runs on threads 4 times by default",
				want: []string{"put/1", "put/1", "put/1", "put/1"},
			},
			{
				name: "runs a case whose section runs on threads n times",
				give: []stateful.Option{stateful.Repeat(2)},
				want: []string{"put/1", "put/1"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				log := &runLog{}
				opts := append([]stateful.Option{stateful.Clients(2)}, tt.give...)
				replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put), opts...) }, repeated...)
				assert.Equal(t, log.all(), tt.want, "a run of client 1's step in each run of the case")
			})
		}

		t.Run("panics for n below 1", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { stateful.Repeat(0) }, "no run")
			assert.Equal(t, got, any("stateful: Repeat(0) is below 1"), "the panic names the option")
		})
	})

	t.Run("Option", func(t *testing.T) {
		t.Parallel()

		t.Run("changes nothing as the zero Option", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			e := replayed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put), stateful.Option{}) }, repeated...)
			assert.Equal(t, choicesOf(e), []uint64{1, 0}, "the defaults' swarm choice and sequential flag")
		})
	})
}

// optionAllocsCases are the cases of the allocation ceilings of the
// options, each kept.
var optionAllocsCases = []alloctest.Case{
	{Name: "Mean", Call: func(assert.TB) { kept = stateful.Mean(1) }, Allocs: optionAllocs},
	{Name: "Max", Call: func(assert.TB) { kept = stateful.Max(1) }, Allocs: optionAllocs},
	{Name: "Swarm", Call: func(assert.TB) { kept = stateful.Swarm(false) }, Allocs: optionAllocs},
	{Name: "Clients", Call: func(assert.TB) { kept = stateful.Clients(2) }, Allocs: optionAllocs},
	{Name: "Concurrent", Call: func(assert.TB) { kept = stateful.Concurrent(1) }, Allocs: optionAllocs},
	{Name: "Tasks", Call: func(assert.TB) { kept = stateful.Tasks(scheduler) }, Allocs: optionAllocs},
	{Name: "Repeat", Call: func(assert.TB) { kept = stateful.Repeat(2) }, Allocs: optionAllocs},
}

// kept is the option that a measured call keeps.
var kept stateful.Option

// scheduler is the scheduler of the measured Tasks.
var scheduler = &stateful.Scheduler{}

// TestOptionAllocs checks that each option allocates its setting alone.
func TestOptionAllocs(t *testing.T) {
	alloctest.Check(t, optionAllocsCases)
}

// BenchmarkOption measures each option, kept.
func BenchmarkOption(b *testing.B) {
	for _, c := range optionAllocsCases {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
