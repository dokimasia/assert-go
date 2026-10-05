// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/enumtest"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// TestPlace checks the parts of a machine's steps, and the step that the
// divergence of a flaky run states: in the run's record, and in the
// sentence that a seat without Report receives.
func TestPlace(t *testing.T) {
	t.Parallel()

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for the six parts and false past them", func(t *testing.T) {
			t.Parallel()
			parts := []prop.Part{
				prop.SwarmPart, prop.SetupPart, prop.SequentialPart,
				prop.ConcurrentPart, prop.DrainPart, prop.SettlePart,
			}
			enumtest.Members(t, parts, []prop.Part{invalidPart})
		})
	})

	t.Run("MarshalText", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the spelling of the part", func(t *testing.T) {
			t.Parallel()
			got, err := prop.SequentialPart.MarshalText()
			assert.NoError(t, err, "every part has a spelling")
			assert.Equal(t, string(got), "sequential", "the spelling of the definition")
		})
	})

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name     string
			give     engine.Place
			wantStep *prop.Place
			wantText string
		}{
			{
				name: "states a swarm choice by the position and the name of its action",
				give: engine.Place{
					Part: engine.SwarmPart, Position: 1, Positioned: true, Action: "get", Acting: true,
				},
				wantStep: &prop.Place{Part: prop.SwarmPart, Position: new(1), Action: new("get")},
				wantText: "the swarm choice 1, of get",
			},
			{
				name:     "states setup without a position",
				give:     engine.Place{Part: engine.SetupPart},
				wantStep: &prop.Place{Part: prop.SetupPart},
				wantText: "setup",
			},
			{
				name:     "states a step that has not chosen its action by its position alone",
				give:     engine.Place{Part: engine.SequentialPart, Position: 4, Positioned: true},
				wantStep: &prop.Place{Part: prop.SequentialPart, Position: new(4)},
				wantText: "sequential step 4, before its action",
			},
			{
				name: "states a step by its position and the name of its action",
				give: engine.Place{
					Part: engine.ConcurrentPart, Position: 0, Positioned: true, Action: "put", Acting: true,
				},
				wantStep: &prop.Place{Part: prop.ConcurrentPart, Position: new(0), Action: new("put")},
				wantText: "concurrent step 0, of put",
			},
			{
				name:     "states the run of a concurrent section without a position",
				give:     engine.Place{Part: engine.ConcurrentPart},
				wantStep: &prop.Place{Part: prop.ConcurrentPart},
				wantText: "the run of the concurrent section",
			},
			{
				name:     "states settle without a position",
				give:     engine.Place{Part: engine.SettlePart},
				wantStep: &prop.Place{Part: prop.SettlePart},
				wantText: "settle",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				body := func() func(*prop.Case) { return placed(tt.give, diverges(prop.Integer(0, 9), prop.Boolean())) }
				want := &prop.Divergence{
					What: prop.RequestDifference, Recorded: "integer in [0, 9]", Replayed: "integer in [0, 1]",
					Label: new(drawn), Step: tt.wantStep,
				}
				assert.Equal(t, detailOf(body(), prop.Seed(7))[divergenceField], any(want), "the divergence")
				seat := &sentences{}
				prop.ForAll(seat, contract, body(), prop.Seed(7))
				sentence := fmt.Sprintf(header, "flaky", 1) +
					fmt.Sprintf("\ndivergence: the request at 0 (in the draw %q, in %s), ", drawn, tt.wantText) +
					"recorded integer in [0, 9], replayed integer in [0, 1]"
				assert.Equal(t, seat.all(), []string{sentence}, "one sentence")
			})
		}
	})
}

// TestPlaceAllocs checks that Valid allocates nothing, and that MarshalText
// allocates its text.
func TestPlaceAllocs(t *testing.T) {
	assert.MaxAllocs(t, func() { _ = prop.SettlePart.Valid() }, 0, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = prop.SettlePart.MarshalText() }, 1, "MarshalText allocates its text")
}

// BenchmarkPlace measures Valid under a ceiling of no allocation, and
// MarshalText.
func BenchmarkPlace(b *testing.B) {
	b.Run("MarshalText", func(b *testing.B) {
		var got []byte
		c := bench.Start(b).MaxAllocs(1)
		defer c.End()
		for c.Loop() {
			got, _ = prop.SettlePart.MarshalText()
		}
		assert.Equal(b, string(got), "settle", "the spelling")
	})

	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = prop.SettlePart.Valid()
		}
		assert.True(b, got, "SettlePart is a part")
	})
}
