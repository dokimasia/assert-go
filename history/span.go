// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/literal"
)

// Span is a call of a history as the record of a check states it: the call
// from its invocation to its completion.
type Span struct {
	// Call is the index of the call's invocation event.
	Call int
	// Completion is the index of the call's completion event, and -1 for a
	// pending call.
	Completion int
	// Process is the process that made the call.
	Process int
	// Operation is the call as the spec sees it.
	Operation Operation
}

// spanJSON is a span in the history's JSON form.
type spanJSON struct {
	Call       int               `json:"call"`
	Completion *int              `json:"completion,omitempty"`
	Process    int               `json:"process"`
	Operation  string            `json:"operation"`
	Args       []json.RawMessage `json:"args"`
	Output     json.RawMessage   `json:"output,omitempty"`
}

// MarshalJSON returns the span in the history's JSON form: the call, the
// completion, the process, the operation, the args as typed literals, and
// the output as a typed literal. A pending call states no completion, and a
// call that is not known states no output. A value that no typed literal
// states is an opaque literal of its text.
//
// # Allocation contract
//
// MarshalJSON allocates the literal of each value, and what encoding/json
// allocates for the span.
func (s Span) MarshalJSON() ([]byte, error) {
	out := spanJSON{Call: s.Call, Process: s.Process, Operation: s.Operation.Name, Args: literals(s.Operation.Args)}
	if s.Completion >= 0 {
		out.Completion = &s.Completion
	}
	if s.Operation.Known {
		out.Output = literal.Detail(s.Operation.Output)
	}
	return json.Marshal(out)
}
