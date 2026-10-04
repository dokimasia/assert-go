// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"go.dokimi.dev/assert/internal/prop/coverage"
	"go.dokimi.dev/assert/internal/prop/engine"
)

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=Verdict -linecomment -output=shortfall.string_gen.go

// Verdict is the coverage test's verdict on a requirement that a run
// missed.
type Verdict uint8

const (
	// Refuted is a requirement whose share the Wilson test showed to be below
	// the required share, with at most one false refutation in 10^9 runs.
	Refuted Verdict = 0 // refuted
	// Unmet is a requirement that the last check, or the exact shares of a
	// domain the run tested in full, left below nine tenths of the required
	// share.
	Unmet Verdict = 1 // unmet
)

// Valid reports whether v is one of the two verdicts.
func (v Verdict) Valid() bool {
	return v <= Unmet
}

// MarshalText returns the verdict's spelling, as the record of a run
// states it.
func (v Verdict) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// Shortfall is a coverage requirement that a run refuted or left unmet,
// with the counts the coverage test decided it from. Its JSON is the
// requirement as the record of a run states it.
type Shortfall struct {
	// Label is the requirement's label, as [Case.Classify] counts it.
	Label string `json:"label"`
	// Share is the share of the valid cases that the label must count, as
	// [Require] states it.
	Share float64 `json:"share"`
	// Counted is the number of valid cases the label counted.
	Counted int `json:"counted"`
	// Valid is the number of valid cases.
	Valid int `json:"valid"`
	// Verdict is the coverage test's verdict.
	Verdict Verdict `json:"verdict"`
}

// shortfallOf returns the engine's shortfall s. The engine reports a
// shortfall only for a refuted or an unmet requirement.
func shortfallOf(s engine.Shortfall) *Shortfall {
	verdict := Refuted
	if s.Verdict == coverage.Unmet {
		verdict = Unmet
	}
	return &Shortfall{
		Label:   s.Requirement.Label,
		Share:   s.Requirement.Share,
		Counted: s.Counted,
		Valid:   s.Valid,
		Verdict: verdict,
	}
}
