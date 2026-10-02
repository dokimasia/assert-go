// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
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
// verdict on the text of a file and the choices it replays.
func TestStore(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		entry := fmt.Sprintf(sessionEntry, sessionFound)
		tests := []struct {
			name string
			give string
			want string
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
				name: "returns an error for a vector that is no JSON object",
				give: `[]`,
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error for recorded choices of no stated kind",
				give: written(`[{}]`, sessionFound, sessionName, entry),
				want: "{} is no choice",
			},
			{
				name: "returns an error for a date of discovery that is no date",
				give: written(`[7]`, "2026-13-01", sessionName, entry),
				want: "month out of range",
			},
			{
				name: "returns an error for another name of the entry's file",
				give: written(`[7]`, sessionFound, "0000000000000000.json", entry),
				want: "the entry's name is " + sessionName,
			},
			{
				name: "returns an error for an entry of another date of discovery",
				give: written(`[7]`, sessionFound, sessionName, fmt.Sprintf(sessionEntry, "2026-10-02")),
				want: "the entry is",
			},
			{
				name: "returns an error for another verdict on a text",
				give: read(entry, "skip", null),
				want: "the verdict is replay, want skip",
			},
			{
				name: "returns an error for replayed choices of no stated kind",
				give: read(entry, "replay", `[{}]`),
				want: "{} is no choice",
			},
			{
				name: "returns an error for replayed choices other than the entry's",
				give: read(entry, "replay", `[8]`),
				want: "the entry replays",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Store, tt.give), tt.want)
			})
		}
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
