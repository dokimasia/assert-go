// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package stateful_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
	"go.dokimi.dev/assert/stateful"
)

// TestTrace checks how the steps of a case follow the step entries of its
// trace, and which entries they refuse.
func TestTrace(t *testing.T) {
	t.Parallel()

	t.Run("Steps", func(t *testing.T) {
		t.Parallel()

		drains := func(log *runLog) stateful.Machine[int] {
			m := machineOf(log, put, get, flush)
			m.Actions[1].Drain, m.Actions[2].Drain = true, true
			return m
		}

		t.Run("keeps the actions that the entries name, and takes each sequential entry as a step", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			got, err := followed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put, get, flush)) },
				step(flush), step(put), step(flush))
			assert.NoError(t, err, "the case takes every entry")
			assert.Equal(t, got, []engine.MachineStep{sequential(flush), sequential(put), sequential(flush)},
				"the steps of the entries")
		})

		t.Run("takes each concurrent entry as a step of its client", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			got, err := followed(func(c *prop.Case) { stateful.Steps(c, machineOf(log, put, get), tasks(c)...) },
				concurrent(get, 2), concurrent(put, 0))
			assert.NoError(t, err, "the case takes every entry")
			assert.Equal(t, got, []engine.MachineStep{{Action: get, Client: 2}, {Action: put, Client: 0}},
				"the steps of the entries")
		})

		t.Run("takes each drain entry as a step of the drain", func(t *testing.T) {
			t.Parallel()
			log := &runLog{}
			got, err := followed(func(c *prop.Case) { stateful.Steps(c, drains(log), stateful.Max(2)) },
				drained(flush), drained(get))
			assert.NoError(t, err, "the case takes every entry")
			want := []engine.MachineStep{
				{Action: flush, Client: -1, Drain: true}, {Action: get, Client: -1, Drain: true},
			}
			assert.Equal(t, got, want, "the steps of the entries")
		})

		tests := []struct {
			name    string
			body    func(c *prop.Case)
			entries []engine.Entry
			want    string
		}{
			{
				name: "refuses a sequential entry of an action that the step does not list",
				body: func(c *prop.Case) {
					m := machineOf(&runLog{}, put, get)
					m.Actions[1].Enabled = func(int) bool { return false }
					stateful.Steps(c, m)
				},
				entries: []engine.Entry{step(put), step(get)},
				want:    `"get" is not among the actions that the step lists`,
			},
			{
				name:    "refuses a sequential entry past the most sequential steps",
				body:    func(c *prop.Case) { stateful.Steps(c, machineOf(&runLog{}, put), stateful.Max(1)) },
				entries: []engine.Entry{step(put), step(put)},
				want:    "the sequential steps are at their maximum of 1",
			},
			{
				name: "refuses a concurrent entry past the most steps of the section",
				body: func(c *prop.Case) {
					stateful.Steps(c, machineOf(&runLog{}, put), append(tasks(c), stateful.Concurrent(1))...)
				},
				entries: []engine.Entry{concurrent(put, 1), concurrent(put, 1)},
				want:    "the concurrent section is at its maximum of 1",
			},
			{
				name:    "refuses a concurrent entry of a client outside the clients",
				body:    func(c *prop.Case) { stateful.Steps(c, machineOf(&runLog{}, put), tasks(c)...) },
				entries: []engine.Entry{concurrent(put, 1), concurrent(put, 3)},
				want:    "client 3 is outside clients 0 to 2",
			},
			{
				name: "refuses a concurrent entry of an action that the section does not list",
				body: func(c *prop.Case) {
					m := machineOf(&runLog{}, put, get)
					m.Actions[1].Enabled = func(int) bool { return true }
					stateful.Steps(c, m, tasks(c)...)
				},
				entries: []engine.Entry{concurrent(put, 1), concurrent(get, 1)},
				want:    `"get" is not among the actions that the step lists`,
			},
			{
				name:    "refuses a concurrent entry of a machine with one client",
				body:    func(c *prop.Case) { stateful.Steps(c, machineOf(&runLog{}, put)) },
				entries: []engine.Entry{step(put), concurrent(put, 1)},
				want:    "the machine runs no concurrent section",
			},
			{
				name:    "refuses a drain entry after the drain",
				body:    func(c *prop.Case) { stateful.Steps(c, drains(&runLog{}), stateful.Max(1)) },
				entries: []engine.Entry{drained(get), drained(get)},
				want:    `"get" is not among the drain actions that are enabled`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := followed(tt.body, tt.entries...)
				f := assert.ErrorAs[*fault.Error](t, err, "a refusal")
				secondStep := fault.Path{fault.Index(1), fault.Field("step")}
				assert.Equal(t, []any{f.Path, f.Reason}, []any{secondStep, tt.want}, "the second entry's step, and why")
			})
		}
	})
}

// followed runs body in the case of entries, and returns the steps that the
// case took, or the refusal of an entry.
func followed(body func(c *prop.Case), entries ...engine.Entry) ([]engine.MachineStep, error) {
	var got []engine.MachineStep
	r := engine.Run(func(c *engine.Case) {
		if _, traced := c.Trace(); traced {
			defer func() { got = stepsOf(engine.Execution{Case: c}) }()
		}
		body((*prop.Case)(c))
	}, engine.Settings{Cases: 1, MaxChoices: engine.MaxChoices, Draws: entries})
	return got, r.Refused
}
