// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"fmt"

	"go.dokimi.dev/assert/internal/prop/engine"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Part -linecomment -output=place.string_gen.go

// Part is a part of the steps of a machine, which the step of a
// [Divergence] names. Each value converts from the engine's part of the
// same value.
type Part uint8

const (
	// SwarmPart is the swarm's choice of the actions that a case keeps.
	SwarmPart Part = 0 // swarm
	// SetupPart is the check and the invariant before the first step.
	SetupPart Part = 1 // setup
	// SequentialPart is the sequential steps.
	SequentialPart Part = 2 // sequential
	// ConcurrentPart is the concurrent section: the steps that it lists, and
	// their run.
	ConcurrentPart Part = 3 // concurrent
	// DrainPart is the steps of the drain.
	DrainPart Part = 4 // drain
	// SettlePart is the settle check and the last invariant.
	SettlePart Part = 5 // settle
)

// Valid reports whether p is one of the six parts.
func (p Part) Valid() bool {
	return p <= SettlePart
}

// MarshalText returns the part's spelling, as the record of a run states
// it.
func (p Part) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// Place is the part of a machine's steps that ran where the replay of a
// flaky run made the request or observed the fingerprint that differed,
// and its step.
type Place struct {
	// Part is the part that ran.
	Part Part `json:"part"`
	// Position is the step's position in its part, from 0, and in the swarm
	// the position of the action whose keep choice it is. It is nil in setup
	// and settle, and while a concurrent section ran the steps that it
	// listed.
	Position *int `json:"position"`
	// Action is the name of the action of the swarm choice or of the step,
	// and nil before the step chose its action and where Position is nil.
	Action *string `json:"action"`
}

// placeOf returns the place of an engine's where, and nil for a where
// outside a machine's steps.
func placeOf(w engine.Where) *Place {
	if !w.Placed {
		return nil
	}
	p := &Place{Part: Part(w.Place.Part)}
	if w.Place.Positioned {
		p.Position = new(w.Place.Position)
	}
	if w.Place.Acting {
		p.Action = new(w.Place.Action)
	}
	return p
}

// text returns the place as the sentence of a run's record states it: the
// part, the step's position, and the step's action or the words that it
// has not chosen one.
func (p Place) text() string {
	if p.Position == nil {
		if p.Part == ConcurrentPart {
			return "the run of the concurrent section"
		}
		return p.Part.String()
	}
	if p.Part == SwarmPart {
		return fmt.Sprintf("the swarm choice %d, of %s", *p.Position, *p.Action)
	}
	step := fmt.Sprintf("%v step %d", p.Part, *p.Position)
	if p.Action == nil {
		return step + ", before its action"
	}
	return step + ", of " + *p.Action
}
