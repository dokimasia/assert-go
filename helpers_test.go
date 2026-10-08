// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"encoding/json"
	"testing"
	"time"
)

// epoch is the instant a controlled clock starts at, chosen so a test
// reading it back cannot pass by accident against a real clock.
var epoch = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// allocContract is the contract of every call of the allocation cases.
const allocContract = "the call passes"

// ledgerKey is the key of a value in the context of a seat, which every
// context that derives from that context returns.
type ledgerKey struct{}

// decoded returns the JSON objects of the call records in lines, and fails
// the test for a line that is no JSON object.
func decoded(t *testing.T, lines []string) []map[string]any {
	t.Helper()
	out := make([]map[string]any, len(lines))
	for i, line := range lines {
		if err := json.Unmarshal([]byte(line), &out[i]); err != nil {
			t.Fatalf("the call record %q is no JSON object: %v", line, err)
		}
	}
	return out
}
