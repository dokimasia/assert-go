// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"strings"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The members of a machines vector that the paths of the tests name.
const (
	setupAt    = "setup"
	strategyAt = "strategy"
	outcomeAt  = "outcome"
)

// The parts of the machines vectors that the tests build.
const (
	// lostOnCrash is the trace of a crash and then a flush of the empty
	// buffer, which the store subject refuses at its second entry.
	lostOnCrash = `[{"step":"crash"},{"step":"flush"}]`
	// flushRefused is the refusal of lostOnCrash.
	flushRefused = `{"entry":1,"name":"flush","reason":"step"}`
	// everyOption is a setup that states each option of a section and the
	// uniform strategy.
	everyOption = `{"mean":5,"max":3,"swarm":false,"clients":3,"concurrent":4,"strategy":"uniform"}`
	// overflowStored is a vector of counter-overflows whose stored case of
	// three increments fails before any case of seed 1, so its run counts no
	// valid case. Without the stored case, the run counts one.
	overflowStored = `{"subject":"counter-overflows","setup":{},` +
		`"settings":{"seed":"1","stored":[[1,0,1,0,1,0,1,0]]},"detail":{"outcome":"counterexample",` +
		`"cases":0,"rejected":0,"seed":"1","counterexample":[{"step":"increment"},{"step":"increment"},` +
		`{"step":"increment"}],"failure":"linearizable","choices":"prop1:AAEAAAABAAAAAQAAAAEAAA",` +
		`"others":[],"divergence":null,"coverage":null},"error":null}`
	// refusingRun is the vector of counter-refuses-every-third whose run is
	// flaky at the index of its fifth sequential step: a case that repeats an
	// earlier case's choices refuses an increment that the earlier case
	// completed, so the step lists two actions where the earlier case listed
	// one.
	refusingRun = `{"subject":"counter-refuses-every-third","setup":{"swarm":false},"settings":{"seed":"1"},` +
		`"detail":{"outcome":"flaky","cases":7,"rejected":0,"seed":"1","counterexample":null,"failure":null,` +
		`"choices":null,"others":null,"divergence":{"what":"request","index":9,` +
		`"recorded":{"kind":"integer","min":0,"max":0},"replayed":{"kind":"integer","min":0,"max":1},` +
		`"label":null,"step":{"part":"sequential","position":4,"action":null}},"coverage":null},"error":null}`
)

// The vectors and the details that the tests derive from others.
var (
	// ninetyNine is passedDetail with 99 valid cases.
	ninetyNine = strings.Replace(passedDetail, "100", "99", 1)
	// unseeded is a vector of correct-queue whose seed is no decimal number.
	unseeded = strings.Replace(machined("correct-queue", `{}`, "", passedDetail, null), `"7"`, `"seven"`, 1)
)

// TestMachines drives the rules of the machines runner that the
// definition's vectors cannot. Written with testing rather than with this
// library, because a verdict is not written with the subject.
func TestMachines(t *testing.T) {
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
				name: "returns nil for a setup that states each option of a section and the uniform strategy",
				give: machined("correct-counter", everyOption, "", passedDetail, null),
			},
			{
				name: "returns nil for a setup that states the PCT strategy",
				give: machined("correct-counter", `{"strategy":{"pct":2}}`, "", passedDetail, null),
			},
			{
				name: "returns nil for a vector whose stored case fails before the cases of its seed",
				give: overflowStored,
			},
			{
				name:       "returns a fault at the subject of a vector that names no machine subject",
				give:       machined("widget", `{}`, "", passedDetail, null),
				wantPath:   inVector(fault.Field(subjectAt)),
				wantReason: `"widget" names no machine subject`,
			},
			{
				name:       "returns a fault at the strategy of a setup that states another name",
				give:       machined("correct-counter", `{"strategy":"random"}`, "", passedDetail, null),
				wantPath:   inVector(fault.Field(setupAt), fault.Field(strategyAt)),
				wantReason: `"random" is no strategy`,
			},
			{
				name:       "returns a fault at the strategy of a setup whose PCT depth is no number",
				give:       machined("correct-counter", `{"strategy":{"pct":"two"}}`, "", passedDetail, null),
				wantPath:   inVector(fault.Field(setupAt), fault.Field(strategyAt)),
				wantReason: `{"pct":"two"} is no strategy`,
			},
			{
				name:       "returns a fault at the strategy of a setup that states an object without a depth",
				give:       machined("correct-counter", `{"strategy":{}}`, "", passedDetail, null),
				wantPath:   inVector(fault.Field(setupAt), fault.Field(strategyAt)),
				wantReason: `{} is no strategy`,
			},
			{
				name:       "returns a fault at the seed of settings that state no decimal number",
				give:       unseeded,
				wantPath:   inVector(fault.Field(settingsAt), fault.Field(seedAt)),
				wantReason: noSeed,
			},
			{
				name:       "returns a fault at the error of a vector that states no refusal of a refused trace",
				give:       machined("store-loses-on-crash", `{}`, lostOnCrash, null, null),
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: "the run refuses an entry, and the vector states none",
			},
			{
				name:       "returns a fault at the error of a vector that states a refusal of a run that refuses none",
				give:       machined("correct-queue", `{"max":1}`, "", passedDetail, flushRefused),
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: `the run refuses no step of entry 1, the step of "flush"`,
			},
			{
				name: "returns a fault at the error of a vector that states the refusal of another entry",
				give: machined("store-loses-on-crash", `{}`, lostOnCrash, null,
					`{"entry":0,"name":"crash","reason":"step"}`),
				wantPath:   inVector(fault.Field(errorAt)),
				wantReason: `the run refuses no step of entry 0, the step of "crash"`,
			},
			{
				name:       "returns a fault at a field of the detail that a passing run states differently",
				give:       machined("correct-queue", `{"max":1}`, "", ninetyNine, null),
				wantPath:   inVector(fault.Field(detailAt), fault.Field(casesAt)),
				wantReason: "the run states 100, want 99",
			},
			{
				name:       "returns a fault at a field of the detail that a failing run states differently",
				give:       machined("counter-overflows", `{}`, "", passedDetail, null),
				wantPath:   inVector(fault.Field(detailAt), fault.Field(outcomeAt)),
				wantReason: `the run states "counterexample", want "passed"`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Machines, tt.give), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns nil again for a second check of a subject that counts over its run", func(t *testing.T) {
			t.Parallel()
			for range 2 {
				expectFault(t, check(t, conformance.Machines, refusingRun), nil, "")
			}
		})
	})
}

// machined returns a machines vector of subject under setup and the
// settings of seed 7, after trace when it is not empty, which states detail
// and error.
func machined(subject, setup, trace, detail, refusal string) string {
	traced := ""
	if trace != "" {
		traced = `"trace":` + trace + ","
	}
	return fmt.Sprintf(`{"subject":%q,"setup":%s,"settings":{"seed":"7"},%s"detail":%s,"error":%s}`,
		subject, setup, traced, detail, refusal)
}
