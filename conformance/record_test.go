// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"strings"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The bodies, requests and draws of the record tests.
const (
	// divergingBody is the body that draws an integer, then a boolean.
	divergingBody = `{"kind":"diverges"}`
	// digitRequest and bitRequest are the requests of divergingBody.
	digitRequest = `{"kind":"integer","min":0,"max":9}`
	bitRequest   = `{"kind":"integer","min":0,"max":1}`
	// unknownDraw is a draw of a typed literal of an unknown type.
	unknownDraw = `{"label":"value","value":` + widget + `}`
	// divergedRun is the divergence of the run of divergingBody under seven,
	// in normal form: in the body's draw, outside a machine's steps.
	divergedRun = `{"index":0,"label":"value","recorded":"integer in [0, 9]","replayed":"integer in [0, 1]",` +
		`"step":null,"what":"request"}`
)

// TestRecord checks the comparison of the record of a run with the detail
// that a behaviour vector states: the first field that differs, in the
// order the definition states the fields, the typed literals of the draws,
// and each side of a divergence, of which a request's bounds compare as
// their text. Written with testing rather than with this library, because
// a verdict is not written with the subject.
func TestRecord(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		at := func(segs ...fault.Segment) fault.Path {
			return inVector(append([]fault.Segment{fault.Field(detailAt)}, segs...)...)
		}
		atRecorded := func(segs ...fault.Segment) fault.Path {
			return at(append([]fault.Segment{fault.Field("divergence"), fault.Field(recordedAt)}, segs...)...)
		}
		atReplayed := func(segs ...fault.Segment) fault.Path {
			return at(append([]fault.Segment{fault.Field("divergence"), fault.Field("replayed")}, segs...)...)
		}
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for a failing run that the vector states",
				give: behaving(bigBody, seven, failedDetail(bigDraw, "big", "[]")),
			},
			{
				name: "returns nil for a divergence whose sides are the requests' bounds",
				give: behaving(divergingBody, seven, diverged(digitRequest, bitRequest)),
			},
			{
				name:       "returns a fault at the failure for a failing run of another failure",
				give:       behaving(bigBody, seven, failedDetail(bigDraw, "small", "[]")),
				wantPath:   at(fault.Field("failure")),
				wantReason: `the run states "big", want "small"`,
			},
			{
				name: "returns a fault at the value of a counterexample of an unknown type",
				give: behaving(bigBody, seven, failedDetail(unknownDraw, "big", "[]")),
				wantPath: at(
					fault.Field("counterexample"),
					fault.Index(0),
					fault.Field(valueAt),
					fault.Field(typeAt),
				),
				wantReason: unknownWidget,
			},
			{
				name: "returns a fault at the nearest passing value of an unknown type",
				give: behaving(bigBody, seven, failedDetail(
					`{"label":"value","value":{"type":"int","value":1001},"nearest-passing":`+widget+`}`, "big", "[]")),
				wantPath: at(fault.Field("counterexample"), fault.Index(0), fault.Field("nearest-passing"),
					fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name: "returns a fault at the value of another failure of an unknown type",
				give: behaving(bigBody, seven, failedDetail(bigDraw, "big",
					`[{"failure":"odd","counterexample":[`+unknownDraw+`],"choices":"prop1:"}]`)),
				wantPath: at(fault.Field("others"), fault.Index(0), fault.Field("counterexample"), fault.Index(0),
					fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at the side for a divergence side of no stated form",
				give:       behaving(divergingBody, seven, diverged(`[1]`, bitRequest)),
				wantPath:   atRecorded(),
				wantReason: "the side is no bounds, fingerprint or identity",
			},
			{
				name:       "returns a fault at the kind for bounds of an unknown kind",
				give:       behaving(divergingBody, seven, diverged(`{"kind":"widget"}`, bitRequest)),
				wantPath:   atRecorded(fault.Field(kindAt)),
				wantReason: `"widget" is no kind of bounds`,
			},
			{
				name: "returns a fault at min for an integer bound of a fractional min",
				give: behaving(
					divergingBody,
					seven,
					diverged(digitRequest, `{"kind":"integer","min":0.5,"max":1}`),
				),
				wantPath:   atReplayed(fault.Field("min")),
				wantReason: "the value is no int",
			},
			{
				name: "returns a fault at max for an integer bound of a fractional max",
				give: behaving(
					divergingBody,
					seven,
					diverged(digitRequest, `{"kind":"integer","min":0,"max":1.5}`),
				),
				wantPath:   atReplayed(fault.Field("max")),
				wantReason: "the value is no int",
			},
			{
				name:       "returns a fault for integer bounds that admit no value",
				give:       behaving(divergingBody, seven, diverged(`{"kind":"integer","min":2,"max":1}`, bitRequest)),
				wantPath:   atRecorded(),
				wantReason: "the integer bounds [2, 1] admit no value",
			},
			{
				name: "returns a fault at min for a float bound of a min of an unknown name",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":"Huge","max":1,"allow_nan":false,"width":64}`, bitRequest)),
				wantPath:   atRecorded(fault.Field("min")),
				wantReason: `"Huge" is none of the names NaN, Inf and -Inf`,
			},
			{
				name: "returns a fault at max for a float bound of a max of an unknown name",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":0,"max":"Large","allow_nan":false,"width":64}`, bitRequest)),
				wantPath:   atRecorded(fault.Field("max")),
				wantReason: `"Large" is none of the names NaN, Inf and -Inf`,
			},
			{
				name: "returns a fault for float bounds that admit no value",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":2,"max":1,"allow_nan":false,"width":64}`, bitRequest)),
				wantPath:   atRecorded(),
				wantReason: "the float bounds [2, 1] admit no value",
			},
			{
				name: "returns a fault for sequence bounds of sizes that admit no size",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"sequence","k":3,"min_size":3,"max_size":1}`, bitRequest)),
				wantPath:   atRecorded(),
				wantReason: "the sizes [3, 1] admit no size",
			},
			{
				name: "returns a fault for sequence bounds of no element",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"sequence","k":0,"min_size":0,"max_size":1}`, bitRequest)),
				wantPath:   atRecorded(),
				wantReason: "the elements [0, 0) admit no value",
			},
			{
				name: "returns a fault that states the float bounds of NaN as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":0,"max":1,"allow_nan":true,"width":64}`, bitRequest)),
				wantPath: at(fault.Field("divergence")),
				wantReason: divergence(`{"index":0,"label":"value","recorded":"float in [0, 1] of width 64 or NaN",` +
					`"replayed":"integer in [0, 1]","step":null,"what":"request"}`),
			},
			{
				name: "returns a fault that states the float bounds without NaN as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"float","min":0,"max":1,"allow_nan":false,"width":32}`, bitRequest)),
				wantPath: at(fault.Field("divergence")),
				wantReason: divergence(`{"index":0,"label":"value","recorded":"float in [0, 1] of width 32",` +
					`"replayed":"integer in [0, 1]","step":null,"what":"request"}`),
			},
			{
				name: "returns a fault that states the bounds of a bounded sequence as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"sequence","k":3,"min_size":0,"max_size":2}`, bitRequest)),
				wantPath: at(fault.Field("divergence")),
				wantReason: divergence(`{"index":0,"label":"value","recorded":"sequence of 0 to 2 values below 3",` +
					`"replayed":"integer in [0, 1]","step":null,"what":"request"}`),
			},
			{
				name: "returns a fault that states the bounds of an unbounded sequence as their text",
				give: behaving(divergingBody, seven,
					diverged(`{"kind":"sequence","k":3,"min_size":1,"max_size":null}`, bitRequest)),
				wantPath: at(fault.Field("divergence")),
				wantReason: divergence(`{"index":0,"label":"value","recorded":"sequence of 1 or more values below 3",` +
					`"replayed":"integer in [0, 1]","step":null,"what":"request"}`),
			},
			{
				name:     "returns a fault that states a fingerprint as its number",
				give:     behaving(divergingBody, seven, diverged(`42`, bitRequest)),
				wantPath: at(fault.Field("divergence")),
				wantReason: divergence(`{"index":0,"label":"value","recorded":42,"replayed":"integer in [0, 1]",` +
					`"step":null,"what":"request"}`),
			},
			{
				name: "returns a fault at the divergence for a divergence in another draw",
				give: behaving(divergingBody, seven,
					strings.Replace(diverged(digitRequest, bitRequest), `"label":"value"`, `"label":"other"`, 1)),
				wantPath: at(fault.Field("divergence")),
				wantReason: divergence(`{"index":0,"label":"other","recorded":"integer in [0, 9]",` +
					`"replayed":"integer in [0, 1]","step":null,"what":"request"}`),
			},
			{
				name: "returns a fault at the divergence for a divergence in a step of a machine",
				give: behaving(divergingBody, seven, strings.Replace(diverged(digitRequest, bitRequest),
					`"step":null`, `"step":{"part":"swarm","position":0,"action":"put"}`, 1)),
				wantPath: at(fault.Field("divergence")),
				wantReason: divergence(`{"index":0,"label":"value","recorded":"integer in [0, 9]",` +
					`"replayed":"integer in [0, 1]","step":{"action":"put","part":"swarm","position":0},` +
					`"what":"request"}`),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Behaviour, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// diverged returns the detail of the run of divergingBody under seven, of
// the divergence of the first request between the sides recorded and
// replayed, each a JSON text, in the body's draw outside a machine's steps.
func diverged(recorded, replayed string) string {
	return fmt.Sprintf(`{"outcome":"flaky","cases":1,"rejected":0,"seed":"7","counterexample":null,`+
		`"failure":null,"choices":null,"others":null,"divergence":{"what":"request","index":0,"recorded":%s,`+
		`"replayed":%s,"label":"value","step":null},"coverage":null}`, recorded, replayed)
}

// divergence returns the reason of a divergence of the run of divergingBody
// that differs from want, the normal form of the one that the vector
// states.
func divergence(want string) string {
	return "the run states " + divergedRun + ", want " + want
}
