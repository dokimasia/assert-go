// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/literal"
)

// Event is one recorded event of a history: an invocation, or the
// completion of the call that an invocation started.
type Event struct {
	// Index is the event's position in recording order, from 0.
	Index int
	// Kind is the event's kind.
	Kind Kind
	// Call is the index of the invocation event of the event's call.
	Call int
	// Client is the client that made the call.
	Client int
	// Process is the process that made the call.
	Process int
	// Operation is the call's operation, on an invocation.
	Operation string
	// Args are the call's arguments, on an invocation.
	Args []any
	// Keys are the keys that the call touches, on an invocation. A call
	// without keys touches every key.
	Keys []any
	// Output is what the call returned, on an [OK] completion.
	Output any
	// Error is the error of a [Fail] or an [Unknown] completion.
	Error error
}

// eventHead is what the JSON form of every event states.
type eventHead struct {
	Index   int  `json:"index"`
	Kind    Kind `json:"kind"`
	Call    int  `json:"call"`
	Client  int  `json:"client"`
	Process int  `json:"process"`
}

// invocationJSON is an invocation in the history's JSON form.
type invocationJSON struct {
	eventHead

	Operation string            `json:"operation"`
	Args      []json.RawMessage `json:"args"`
	Keys      []json.RawMessage `json:"keys"`
}

// okJSON is an ok completion in the history's JSON form.
type okJSON struct {
	eventHead

	Output json.RawMessage `json:"output"`
}

// errorJSON is a fail or an unknown completion in the history's JSON form.
type errorJSON struct {
	eventHead

	Error string `json:"error"`
}

// MarshalJSON returns the event in the history's JSON form: the index, the
// kind, the call, the client and the process, and then the operation, the
// args and the keys of an invocation, the output of an ok completion, or
// the error's text of a fail or an unknown one. The args, the keys and the
// output are typed literals, and a value that no typed literal states is an
// opaque literal of its text. A nil error states the empty text.
//
// # Allocation contract
//
// MarshalJSON allocates the literal of each value, and what encoding/json
// allocates for the event.
func (e Event) MarshalJSON() ([]byte, error) {
	head := eventHead{Index: e.Index, Kind: e.Kind, Call: e.Call, Client: e.Client, Process: e.Process}
	switch e.Kind {
	case Invoke:
		return json.Marshal(invocationJSON{
			eventHead: head, Operation: e.Operation, Args: literals(e.Args), Keys: literals(e.Keys),
		})
	case OK:
		return json.Marshal(okJSON{eventHead: head, Output: literal.Detail(e.Output)})
	}
	text := ""
	if e.Error != nil {
		text = e.Error.Error()
	}
	return json.Marshal(errorJSON{eventHead: head, Error: text})
}
