// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"encoding/json"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// stepJSONAllocs are the allocations of MarshalJSON on a step of a
// concurrent section, measured.
const stepJSONAllocs = 4

// TestStep checks the JSON of a step of a counterexample, and its line in
// the sentence of a run's record.
func TestStep(t *testing.T) {
	t.Parallel()

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give prop.Step
			want string
		}{
			{
				name: "returns the action of a sequential step",
				give: prop.Step{Action: "put", Client: -1},
				want: `{"step":"put"}`,
			},
			{
				name: "returns the client of a step of a concurrent section, client 0 included",
				give: prop.Step{Action: "put", Client: 0},
				want: `{"step":"put","client":0}`,
			},
			{
				name: "returns the drain mark of a step of the drain",
				give: prop.Step{Action: "deliver", Client: -1, Drain: true},
				want: `{"step":"deliver","drain":true}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := json.Marshal(tt.give)
				assert.NoError(t, err, "the step is JSON")
				assert.Equal(t, string(got), tt.want, "the step as the record of a run states it")
			})
		}
	})

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		t.Run("states each step in the sentence of the record, with its client or the drain", func(t *testing.T) {
			t.Parallel()
			seat := &sentences{}
			prop.ForAll(seat, contract, func(c *prop.Case) {
				machine := (*engine.Case)(c)
				machine.Step(engine.MachineStep{Action: "put", Client: -1})
				machine.Step(engine.MachineStep{Action: "get", Client: 1})
				machine.Step(engine.MachineStep{Action: "deliver", Client: -1, Drain: true})
				c.Rand().Uint64()
				fail(c, always)
			}, prop.Seed(7))
			all := seat.all()
			assert.Length(t, all, 1, "one sentence")
			assert.Contains(t, all[0], "\n  step put\n  step get on client 1\n  step deliver in the drain",
				"the three steps")
		})
	})
}

// TestStepAllocs checks the ceiling of MarshalJSON.
func TestStepAllocs(t *testing.T) {
	s := prop.Step{Action: "put", Client: 1}
	assert.MaxAllocs(t, func() { _, _ = s.MarshalJSON() }, stepJSONAllocs, "MarshalJSON allocates its JSON")
}

// BenchmarkStep measures MarshalJSON.
func BenchmarkStep(b *testing.B) {
	b.Run("MarshalJSON", func(b *testing.B) {
		s := prop.Step{Action: "put", Client: 1}
		got, _ := s.MarshalJSON()
		c := bench.Start(b).MaxAllocs(stepJSONAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = s.MarshalJSON()
		}
		assert.Equal(b, string(got), `{"step":"put","client":1}`, "the step and its client")
	})
}
