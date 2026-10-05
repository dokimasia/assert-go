// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/fault"
)

// isolationContract is the contract of the check of an isolation vector.
const isolationContract = "the transactions of the vector are isolated"

// checkSerializable checks the history of a serializable vector with
// [history.Serializable], as checkIsolation states.
func checkSerializable(raw json.RawMessage, _ string) error {
	return checkIsolation(raw, history.Serializable)
}

// checkSnapshotIsolation checks the history of a snapshot-isolation vector
// with [history.HasSnapshotIsolation], as checkIsolation states.
func checkSnapshotIsolation(raw json.RawMessage, _ string) error {
	return checkIsolation(raw, history.HasSnapshotIsolation)
}

// checkIsolation records the history of an isolation vector, checks it with
// level on a recorder, and compares the verdict with the one that the vector
// states. A failing vector compares each field of the detail of the call
// record with the vector's.
func checkIsolation(raw json.RawMessage, level func(assert.TB, *history.History, string)) error {
	var v struct {
		History []json.RawMessage          `json:"history"`
		Expect  string                     `json:"expect"`
		Detail  map[string]json.RawMessage `json:"detail"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	h, err := historyOf(v.History)
	if err != nil {
		return err
	}
	rec := assert.NewRecorder()
	level(rec, h, isolationContract)
	// A recorder keeps the call record of every call.
	c := callsOf(rec)[0]
	switch v.Expect {
	case expectFail:
		return compareFailure(c, v.Detail)
	case expectPass:
		if c.Verdict != expectPass {
			return fault.At(fault.New("the check ends as %s, want pass", c.Verdict), fault.Field(expectMember))
		}
		return nil
	}
	return fault.At(fault.New("the vector expects %q, neither pass nor fail", v.Expect), fault.Field(expectMember))
}
