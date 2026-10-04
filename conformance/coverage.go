// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/coverage"
)

// checkCoverage decides the requirement of a coverage vector at its check,
// and compares the verdict. The exact shares of a domain tested in full
// decide over the last check.
func checkCoverage(raw json.RawMessage, _ string) error {
	var v struct {
		// Counted is the number of valid cases the label counted.
		Counted int `json:"counted"`
		// Valid is the number of valid cases.
		Valid int `json:"valid"`
		// Share is the required share.
		Share float64 `json:"share"`
		// Last reports whether the check is the last.
		Last bool `json:"last"`
		// Exact reports whether the run tested every input.
		Exact bool `json:"exact"`
		// Verdict is the verdict's spelling.
		Verdict string `json:"verdict"`
	}
	if err := decode(raw, &v); err != nil {
		return err
	}
	stage := coverage.Interim
	if v.Last {
		stage = coverage.Final
	}
	if v.Exact {
		stage = coverage.Exhausted
	}
	if got := coverage.Decide(v.Counted, v.Valid, v.Share, stage); got.String() != v.Verdict {
		return fault.At(fault.New("the verdict is %v, want %s", got, v.Verdict), fault.Field(verdictMember))
	}
	return nil
}
