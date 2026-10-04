// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The definition's vector
// store/records-an-assertion-without-a-location-by-its-contract.
const (
	// sessionContract is the contract of its property.
	sessionContract = "a stored session expires"
	// sessionName is the name of its entry's file.
	sessionName = "68df0f22b1940653.json"
	// sessionFound is its date of discovery.
	sessionFound = "2026-10-01"
	// sessionEntry is its entry's JSON, of the date of discovery as a verb.
	sessionEntry = `{"store":1,"definition":"1.2.0","property":"a stored session expires",` +
		`"identity":{"assertion":"equal","contract":"the session is gone"},"choices":"prop1:AAc",` +
		`"counterexample":[{"label":"ttl","value":{"type":"int","value":7}}],"found":%q}`
)

// TestStore checks a store vector: the entry that a failure writes, or the
// verdict on the text of a file and the choices it replays. Written with
// testing rather than with this library, because a verdict is not written
// with the subject.
func TestStore(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		entry := fmt.Sprintf(sessionEntry, sessionFound)
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for a vector that states the written entry",
				give: written(`[7]`, sessionFound, sessionName, entry),
			},
			{
				name: "returns nil for a vector that states the replayed choices of a text",
				give: read(entry, "replay", `[7]`),
			},
			{
				name: "returns nil for a vector that states a text as damaged",
				give: read("store: 1", "damaged", null),
			},
			{
				name:       "returns a fault at the index of a recorded choice of no stated kind",
				give:       written(`[{}]`, sessionFound, sessionName, entry),
				wantPath:   inVector(fault.Field(choicesAt), fault.Index(0)),
				wantReason: "{} is no choice",
			},
			{
				name:       "returns a fault at the name for another name of the entry's file",
				give:       written(`[7]`, sessionFound, "0000000000000000.json", entry),
				wantPath:   inVector(fault.Field("name")),
				wantReason: "the entry's name is " + sessionName + ", want 0000000000000000.json",
			},
			{
				name:     "returns a fault at the entry for an entry of another date of discovery",
				give:     written(`[7]`, sessionFound, sessionName, fmt.Sprintf(sessionEntry, "2026-10-02")),
				wantPath: inVector(fault.Field("entry")),
				wantReason: "the entry is " + fmt.Sprintf(sessionEntry, sessionFound) +
					", want " + fmt.Sprintf(sessionEntry, "2026-10-02"),
			},
			{
				name:       "returns a fault at the verdict for another verdict on a text",
				give:       read(entry, "skip", null),
				wantPath:   inVector(fault.Field("verdict")),
				wantReason: "the verdict is replay, want skip",
			},
			{
				name:       "returns a fault at the index of a replayed choice of no stated kind",
				give:       read(entry, "replay", `[{}]`),
				wantPath:   inVector(fault.Field(choicesAt), fault.Index(0)),
				wantReason: "{} is no choice",
			},
			{
				name:       "returns a fault at the choices for replayed choices other than the entry's",
				give:       read(entry, "replay", `[8]`),
				wantPath:   inVector(fault.Field(choicesAt)),
				wantReason: "the choices are prop1:AAc, want [8]",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Store, tt.give), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns a fault at found for a date of discovery that is no date", func(t *testing.T) {
			t.Parallel()
			err := check(t, conformance.Store, written(`[7]`, "2026-13-01", sessionName, entry))
			expectFault(t, err, inVector(fault.Field("found")), `"2026-13-01" is no date`)
			if _, ok := errors.AsType[*time.ParseError](err); !ok {
				t.Fatalf("Check returns %v, want one caused by the parse of the date", err)
			}
		})
	})
}

// written returns a store vector that writes the entry of the session
// property for choices, found on the date found, as the file name of the
// JSON entry.
func written(choices, found, name, entry string) string {
	return fmt.Sprintf(`{"contract":%q,"choices":%s,`+
		`"identity":{"assertion":"equal","contract":"the session is gone"},`+
		`"counterexample":[{"label":"ttl","value":{"type":"int","value":7}}],`+
		`"definition":"1.2.0","found":%q,"name":%q,"entry":%s}`,
		sessionContract, choices, found, name, entry)
}

// read returns a store vector that reads text for the session property, and
// states the verdict and the choices to replay.
func read(text, verdict, choices string) string {
	return fmt.Sprintf(`{"contract":%q,"text":%q,"verdict":%q,"choices":%s}`, sessionContract, text, verdict, choices)
}
