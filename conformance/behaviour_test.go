// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// TestBehaviour checks a behaviour vector: the run of a body under its
// settings, with its stored cases and its replay, and the record of the
// run. Written with testing rather than with this library, because a
// verdict is not written with the subject.
func TestBehaviour(t *testing.T) {
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
				name: "returns nil for a replay of 1000 that fails-once passes",
				give: behaving(`{"kind":"fails-once"}`, `{"seed":"7","replay":"prop1:AOgH"}`,
					`{"outcome":"passed","cases":1,"rejected":0,"seed":"7","counterexample":null,`+
						`"failure":null,"choices":null,"others":null,"divergence":null,"coverage":null}`),
			},
			{
				name: "returns nil for a passing run of the stated number of cases",
				give: behaving(passingBody, `{"seed":"7","cases":10}`,
					`{"outcome":"passed","cases":10,"rejected":0,"seed":"7","counterexample":null,`+
						`"failure":null,"choices":null,"others":null,"divergence":null,"coverage":null}`),
			},
			{
				name:       "returns a fault at the stored case for stored choices of no stated kind",
				give:       behaving(passingBody, `{"seed":"7","stored":[[7],[{}]]}`, passedDetail),
				wantPath:   inVector(fault.Field(settingsAt), fault.Field("stored"), fault.Index(1), fault.Index(0)),
				wantReason: "{} is no choice",
			},
			{
				name:       "returns a fault at the replay for a replay token without the prefix",
				give:       behaving(passingBody, `{"seed":"7","replay":"AAc"}`, passedDetail),
				wantPath:   inVector(fault.Field(settingsAt), fault.Field("replay")),
				wantReason: `"AAc" does not start with prop1:`,
			},
			{
				name:       "returns a fault at the outcome for a passing run that the vector states as failing",
				give:       behaving(passingBody, seven, failedDetail(bigDraw, "big", "[]")),
				wantPath:   inVector(fault.Field(detailAt), fault.Field("outcome")),
				wantReason: `the run states "passed", want "counterexample"`,
			},
			{
				name: "returns a fault at the draw of a passing run's vector that states no typed literal of it",
				give: behaving(passingBody, seven,
					`{"outcome":"passed","cases":100,"rejected":0,"seed":"7","counterexample":[{"label":"value",`+
						`"value":`+widget+`}],"failure":null,"choices":null,"others":null,"divergence":null,`+
						`"coverage":null}`),
				wantPath: inVector(fault.Field(detailAt), fault.Field("counterexample"), fault.Index(0),
					fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name: "returns a fault at the cases for a passing run of another number of cases",
				give: behaving(passingBody, seven,
					`{"outcome":"passed","cases":99,"rejected":0,"seed":"7","counterexample":null,`+
						`"failure":null,"choices":null,"others":null,"divergence":null,"coverage":null}`),
				wantPath:   inVector(fault.Field(detailAt), fault.Field(casesAt)),
				wantReason: "the run states 100, want 99",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Behaviour, tt.give), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns a fault at the stored case for a store directory that is a file", func(t *testing.T) {
			t.Parallel()
			file := filepath.Join(t.TempDir(), "store")
			if err := os.WriteFile(file, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			raw := behaving(bigBody, `{"seed":"7","stored":[[950]]}`, passedDetail)
			v := conformance.Vector{Kind: conformance.Behaviour, ID: builtID, Raw: json.RawMessage(raw)}
			expectFault(t, v.Check(file), inVector(fault.Field(settingsAt), fault.Field("stored"), fault.Index(0)),
				"the store cannot be created")
		})
	})
}
