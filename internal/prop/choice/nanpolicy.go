// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package choice

//go:generate go run golang.org/x/tools/cmd/stringer@v0.50.0 -type=NaNPolicy -linecomment -output=nanpolicy.string_gen.go

// NaNPolicy states whether NaN is a value of [FloatBounds].
type NaNPolicy uint8

const (
	// ExcludeNaN leaves NaN out of the bounds, so a recorded NaN replays
	// as the target.
	ExcludeNaN NaNPolicy = 0 // exclude-nan
	// AdmitNaN makes NaN a value of the bounds, as [NaN].
	AdmitNaN NaNPolicy = 1 // admit-nan
)

// Valid reports whether p is one of the two policies.
func (p NaNPolicy) Valid() bool {
	return p <= AdmitNaN
}
