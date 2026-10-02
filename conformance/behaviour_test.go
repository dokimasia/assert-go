// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// The bodies, settings and details of the definition's behaviour vectors
// that the tests vary.
const (
	// seven are the settings of a run of seed 7.
	seven = `{"seed":"7"}`
	// passingBody draws an integer in [0, 1000] and never fails.
	passingBody = `{"draw":{"gen":"integer","min":0,"max":1000}}`
	// passedDetail is the detail of the run of passingBody under seven.
	passedDetail = `{"outcome":"passed","cases":100,"rejected":0,"seed":"7","counterexample":null,` +
		`"failure":null,"choices":null,"others":null,"divergence":null,"coverage":null}`
	// bigBody draws an integer in [0, 10000] and fails as big from 1001.
	bigBody = `{"draw":{"gen":"integer","min":0,"max":10000},` +
		`"fails":[{"identity":"big","when":{"kind":"at-least","n":1001}}]}`
	// bigDraw is the draw of the minimal counterexample of bigBody.
	bigDraw = `{"label":"value","value":{"type":"int","value":1001},"any-value-fails":false,` +
		`"nearest-passing":{"type":"int","value":1000}}`
	// divergingBody is the body that draws an integer, then a boolean.
	divergingBody = `{"kind":"diverges"}`
	// digitRequest and bitRequest are the requests of divergingBody.
	digitRequest = `{"kind":"integer","min":0,"max":9}`
	bitRequest   = `{"kind":"integer","min":0,"max":1}`
	// unknownDraw is a draw of a typed literal of an unknown type.
	unknownDraw = `{"label":"value","value":{"type":"widget"}}`
)

// TestBehaviour checks a behaviour vector: the record of a body's run under
// its settings.
func TestBehaviour(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns an error for a vector that is no JSON object",
				give: `[]`,
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error for a seed that is no decimal number",
				give: behaving(passingBody, `{"seed":"seven"}`, passedDetail),
				want: "invalid syntax",
			},
			{
				name: "returns an error for stored choices of no stated kind",
				give: behaving(passingBody, `{"seed":"7","stored":[[{}]]}`, passedDetail),
				want: "{} is no choice",
			},
			{
				name: "returns an error for a replay token without the prefix",
				give: behaving(passingBody, `{"seed":"7","replay":"AAc"}`, passedDetail),
				want: "does not start with prop1:",
			},
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
				name: "returns an error for a passing run that the vector states as failing",
				give: behaving(passingBody, seven, failedDetail(bigDraw, "big", "[]")),
				want: "the run passes, want counterexample",
			},
			{
				name: "returns an error for a passing run of another number of cases",
				give: behaving(passingBody, seven,
					`{"outcome":"passed","cases":99,"rejected":0,"seed":"7","counterexample":null,`+
						`"failure":null,"choices":null,"others":null,"divergence":null,"coverage":null}`),
				want: "the run ends as [passed 100 0 7], want [passed 99 0 7]",
			},
			{
				name: "returns an error for a failing run of another failure",
				give: behaving(bigBody, seven, failedDetail(bigDraw, "small", "[]")),
				want: "the record's detail is",
			},
			{
				name: "returns an error for a counterexample value of an unknown type",
				give: behaving(bigBody, seven, failedDetail(unknownDraw, "big", "[]")),
				want: conformance.ErrUnknownType.Error(),
			},
			{
				name: "returns an error for a nearest passing value of an unknown type",
				give: behaving(bigBody, seven, failedDetail(
					`{"label":"value","value":{"type":"int","value":1001},"nearest-passing":{"type":"widget"}}`,
					"big", "[]")),
				want: conformance.ErrUnknownType.Error(),
			},
			{
				name: "returns an error for another failure's value of an unknown type",
				give: behaving(bigBody, seven, failedDetail(bigDraw, "big",
					`[{"failure":"odd","counterexample":[`+unknownDraw+`],"choices":"prop1:"}]`)),
				want: conformance.ErrUnknownType.Error(),
			},
			{
				name: "returns an error for a divergence side of no stated form",
				give: behaving(divergingBody, seven, diverged(`[1]`, bitRequest)),
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error for an integer bound of a fractional min",
				give: behaving(divergingBody, seven, diverged(digitRequest, `{"kind":"integer","min":0.5,"max":1}`)),
				want: "decode scalar",
			},
			{
				name: "returns an error for an integer bound of a fractional max",
				give: behaving(divergingBody, seven, diverged(digitRequest, `{"kind":"integer","min":0,"max":1.5}`)),
				want: "decode scalar",
			},
			{
				name: "returns an error for a float bound of a min of an unknown name",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":"Huge","max":1,"allow_nan":false,"width":64}`, bitRequest)),
				want: `unrecognized float literal "Huge"`,
			},
			{
				name: "returns an error for a float bound of a max of an unknown name",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":0,"max":"Large","allow_nan":false,"width":64}`, bitRequest)),
				want: `unrecognized float literal "Large"`,
			},
			{
				name: "returns an error that states the float bounds of NaN as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":0,"max":1,"allow_nan":true,"width":64}`, bitRequest)),
				want: `"recorded":"float in [0, 1] of width 64 or NaN"`,
			},
			{
				name: "returns an error that states the float bounds without NaN as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":0,"max":1,"allow_nan":false,"width":32}`, bitRequest)),
				want: `"recorded":"float in [0, 1] of width 32"`,
			},
			{
				name: "returns an error that states the bounds of a bounded sequence as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"sequence","k":3,"min_size":0,"max_size":2}`, bitRequest)),
				want: `"recorded":"sequence of 0 to 2 values below 3"`,
			},
			{
				name: "returns an error that states the bounds of an unbounded sequence as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"sequence","k":3,"min_size":1,"max_size":null}`, bitRequest)),
				want: `"recorded":"sequence of 1 or more values below 3"`,
			},
			{
				name: "returns an error that states a fingerprint as its number",
				give: behaving(divergingBody, seven, diverged(`42`, bitRequest)),
				want: `"recorded":42`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Behaviour, tt.give), tt.want)
			})
		}

		t.Run("returns an error for a store directory that is a file", func(t *testing.T) {
			t.Parallel()
			file := filepath.Join(t.TempDir(), "store")
			if err := os.WriteFile(file, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			raw := behaving(bigBody, `{"seed":"7","stored":[[950]]}`, passedDetail)
			v := conformance.Vector{Kind: conformance.Behaviour, ID: builtID, Raw: json.RawMessage(raw)}
			expectCheck(t, v.Check(file), "not a directory")
		})
	})
}

// behaving returns a behaviour vector of the run of body under settings,
// which states detail.
func behaving(body, settings, detail string) string {
	return fmt.Sprintf(`{"body":%s,"settings":%s,"detail":%s}`, body, settings, detail)
}

// failedDetail returns the detail of the run of bigBody under seven, of the
// minimal counterexample's one draw, the failure's identity, and the other
// failures, each a JSON text.
func failedDetail(draw, failure, others string) string {
	return fmt.Sprintf(`{"outcome":"counterexample","cases":1,"rejected":0,"seed":"7","counterexample":[%s],`+
		`"failure":%q,"choices":"prop1:AOkH","others":%s,"divergence":null,"coverage":null}`, draw, failure, others)
}

// diverged returns the detail of the run of divergingBody under seven, of
// the divergence of the first request between the sides recorded and
// replayed, each a JSON text.
func diverged(recorded, replayed string) string {
	return fmt.Sprintf(`{"outcome":"flaky","cases":1,"rejected":0,"seed":"7","counterexample":null,`+
		`"failure":null,"choices":null,"others":null,`+
		`"divergence":{"what":"request","index":0,"recorded":%s,"replayed":%s},"coverage":null}`, recorded, replayed)
}
