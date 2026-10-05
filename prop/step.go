// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Step is one step of a machine in a counterexample, an [Entry]: the action
// that it took, the client that ran it, and whether the drain took it.
type Step struct {
	// Action is the name of the step's action.
	Action string
	// Client is the client of a step of a concurrent section, and -1 for any
	// other step.
	Client int
	// Drain reports a step of the drain.
	Drain bool
}

// stepJSON is a step as the record of a run states it.
type stepJSON struct {
	Step   string `json:"step"`
	Client *int   `json:"client,omitempty"`
	Drain  bool   `json:"drain,omitempty"`
}

// MarshalJSON returns the step as the record of a run states a step of its
// counterexample: the action under "step", the client under "client" for a
// step of a concurrent section, and "drain": true for a step of the drain.
//
// # Allocation contract
//
// MarshalJSON allocates four times for a step of a concurrent section.
func (s Step) MarshalJSON() ([]byte, error) {
	out := stepJSON{Step: s.Action, Drain: s.Drain}
	if s.Client >= 0 {
		out.Client = &s.Client
	}
	return json.Marshal(out)
}

// writeLine writes the step's line: its action, and its client or the
// drain.
func (s Step) writeLine(b *strings.Builder) {
	fmt.Fprintf(b, "\n  step %s", s.Action)
	if s.Client >= 0 {
		fmt.Fprintf(b, " on client %d", s.Client)
	}
	if s.Drain {
		b.WriteString(" in the drain")
	}
}

// brief returns the step as it is: a step states no explanation.
func (s Step) brief() any {
	return s
}
