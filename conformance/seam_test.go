// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"strings"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The members of a seam vector that the paths of the tests name.
const (
	scriptAt    = "script"
	intervalsAt = "intervals"
	eventsAt    = "events"
	refusedAt   = "refused"
	keysAt      = "keys"
	outputAt    = "output"
)

// The JSON texts of the parts of a seam vector that several tests build.
const (
	// nullLiteral is the typed literal of null.
	nullLiteral = `{"type":"null"}`
	// readEvent is the JSON form of the invocation of read by client 0 at
	// index 0.
	readEvent = `{"index":0,"kind":"invoke","call":0,"client":0,"process":0,"operation":"read","args":[],"keys":[]}`
	// noCompletion is the reason of an interval entry that states an end
	// without a completion kind, or the reverse.
	noCompletion = "the entry states a completion kind and an end, or neither"
)

// TestSeam drives the rules of the seam runner that the definition's
// vectors cannot. Written with testing rather than with this library,
// because a verdict is not written with the subject.
func TestSeam(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault for a vector of neither a script nor intervals",
				give:       `{"events":[],"refused":null}`,
				wantPath:   inVector(),
				wantReason: "the vector states a script or intervals",
			},
			{
				name:       "returns a fault for a vector of a script and intervals",
				give:       `{"script":[],"intervals":[],"events":[],"refused":null}`,
				wantPath:   inVector(),
				wantReason: "the vector states a script or intervals",
			},
			{
				name:       "returns a fault at refused for a script whose refused entry the vector does not state",
				give:       script(`null`, `null`, invoking(0, 0), invoking(1, 0)),
				wantPath:   inVector(fault.Field(refusedAt)),
				wantReason: "the seam refuses entry 1, want null",
			},
			{
				name:       "returns a fault at refused for a script that the seam does not refuse",
				give:       script(`null`, `1`, invoking(0, 0)),
				wantPath:   inVector(fault.Field(refusedAt)),
				wantReason: "the seam refuses no entry, want entry 1",
			},
			{
				name:       "returns a fault at an event that differs from the vector's",
				give:       script(`[`+readEvent+`]`, `null`, invoking(0, 1)),
				wantPath:   inVector(fault.Field(eventsAt), fault.Index(0)),
				wantReason: "the event is " + clientOf(1) + ", want " + readEvent,
			},
			{
				name:       "returns a fault at the events for a history of more events than the vector's",
				give:       script(`[]`, `null`, invoking(0, 0)),
				wantPath:   inVector(fault.Field(eventsAt)),
				wantReason: "the history records 1 events, want 0",
			},
			{
				name:       "returns a fault at a script entry that is no JSON object",
				give:       script(`[]`, `null`, `"x"`),
				wantPath:   inVector(fault.Field(scriptAt), fault.Index(0)),
				wantReason: notParsed,
			},
			{
				name: "returns a fault at an argument of an invocation that is no typed literal",
				give: script(`[]`, `null`, `{"invoke":0,"client":0,"operation":"write","args":[`+widget+`],"keys":[]}`),
				wantPath: inVector(fault.Field(scriptAt), fault.Index(0), fault.Field(argsAt), fault.Index(0),
					fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name: "returns a fault at a key of an invocation that is no typed literal",
				give: script(`[]`, `null`, `{"invoke":0,"client":0,"operation":"read","args":[],"keys":[`+widget+`]}`),
				wantPath: inVector(fault.Field(scriptAt), fault.Index(0), fault.Field(keysAt), fault.Index(0),
					fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at an output of a completion that is no typed literal",
				give:       script(`[]`, `null`, invoking(0, 0), `{"ok":0,"output":`+widget+`}`),
				wantPath:   inVector(fault.Field(scriptAt), fault.Index(1), fault.Field(outputAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at a completion of a call that the script does not open",
				give:       script(`[]`, `null`, `{"fail":5,"error":"refused"}`),
				wantPath:   inVector(fault.Field(scriptAt), fault.Index(0)),
				wantReason: "the entry completes call 5, which the script does not open",
			},
			{
				name:       "returns a fault at a script entry that states no call of the seam",
				give:       script(`[]`, `null`, `{}`),
				wantPath:   inVector(fault.Field(scriptAt), fault.Index(0)),
				wantReason: "the entry states no call of the seam",
			},
			{
				name:       "returns a fault at an interval entry of an end without a completion kind",
				give:       intervals(`null`, `{"client":0,"operation":"read","args":[],"keys":[],"start":0,"end":1}`),
				wantPath:   inVector(fault.Field(intervalsAt), fault.Index(0)),
				wantReason: noCompletion,
			},
			{
				name: "returns a fault at an interval entry of a completion kind without an end",
				give: intervals(
					`null`,
					`{"client":0,"operation":"read","args":[],"keys":[],"start":0,"kind":"ok","output":`+nullLiteral+`}`,
				),
				wantPath:   inVector(fault.Field(intervalsAt), fault.Index(0)),
				wantReason: noCompletion,
			},
			{
				name: "returns a fault at an interval entry of a kind that completes no call",
				give: intervals(`null`,
					`{"client":0,"operation":"read","args":[],"keys":[],"start":0,"end":1,"kind":"invoke"}`),
				wantPath:   inVector(fault.Field(intervalsAt), fault.Index(0)),
				wantReason: noCompletion,
			},
			{
				name: "returns a fault at an argument of an interval entry that is no typed literal",
				give: intervals(`null`, `{"client":0,"operation":"write","args":[`+widget+`],"keys":[],"start":0}`),
				wantPath: inVector(fault.Field(intervalsAt), fault.Index(0), fault.Field(argsAt), fault.Index(0),
					fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name: "returns a fault at a key of an interval entry that is no typed literal",
				give: intervals(`null`, `{"client":0,"operation":"read","args":[],"keys":[`+widget+`],"start":0}`),
				wantPath: inVector(fault.Field(intervalsAt), fault.Index(0), fault.Field(keysAt), fault.Index(0),
					fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name: "returns a fault at an output of an interval entry that is no typed literal",
				give: intervals(
					`null`,
					`{"client":0,"operation":"read","args":[],"keys":[],"start":0,"end":1,"kind":"ok","output":`+widget+`}`,
				),
				wantPath: inVector(
					fault.Field(intervalsAt),
					fault.Index(0),
					fault.Field(outputAt),
					fault.Field(typeAt),
				),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at refused for intervals that from-intervals accepts",
				give:       intervals(`0`, interval(0, 0)),
				wantPath:   inVector(fault.Field(refusedAt)),
				wantReason: "from-intervals returns <nil>, want a refusal of 0",
			},
			{
				name:     "returns a fault at refused for intervals whose refusal the vector does not state",
				give:     intervals(`null`, interval(0, 2)),
				wantPath: inVector(fault.Field(refusedAt)),
				wantReason: "from-intervals returns history.FromIntervals: [0]: ends at 1, before it starts at 2, " +
					"want a refusal of null",
			},
			{
				name:     "returns a fault at refused for intervals that from-intervals refuses at another entry",
				give:     intervals(`1`, interval(0, 2), interval(1, 0)),
				wantPath: inVector(fault.Field(refusedAt)),
				wantReason: "from-intervals returns history.FromIntervals: [0]: ends at 1, before it starts at 2, " +
					"want a refusal of 1",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Seam, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// invoking returns the script entry of an invocation of read without keys
// under the number n by client.
func invoking(n, client int) string {
	return fmt.Sprintf(`{"invoke":%d,"client":%d,"operation":"read","args":[],"keys":[]}`, n, client)
}

// clientOf returns the JSON form of the invocation of read by client at
// index 0, on the process 0.
func clientOf(client int) string {
	return fmt.Sprintf(`{"index":0,"kind":"invoke","call":0,"client":%d,"process":0,"operation":"read",`+
		`"args":[],"keys":[]}`, client)
}

// interval returns the interval entry of an ok read by client from start to
// the time 1, whose output is null.
func interval(client, start int) string {
	return fmt.Sprintf(`{"client":%d,"operation":"read","args":[],"keys":[],"start":%d,"end":1,"kind":"ok",`+
		`"output":`+nullLiteral+`}`, client, start)
}

// script returns a seam vector of the script entries, which states events
// and refused, each a JSON text.
func script(events, refused string, entries ...string) string {
	return fmt.Sprintf(`{"script":[%s],"events":%s,"refused":%s}`, strings.Join(entries, ","), events, refused)
}

// intervals returns a seam vector of the interval entries, which states no
// events and refused, a JSON text.
func intervals(refused string, entries ...string) string {
	return fmt.Sprintf(`{"intervals":[%s],"events":null,"refused":%s}`, strings.Join(entries, ","), refused)
}
