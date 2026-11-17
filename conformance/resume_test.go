// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The members of a resume vector that the paths of the tests name.
const (
	checksAt    = "checks"
	resultsAt   = "results"
	nextCallsAt = "next-calls"
)

// The checks and the results of a resume vector of readsOne that several
// tests build. The first check reads the write, and the second check
// continues its search over the read.
const (
	// readsOneChecks are the two checks, after the write and after the read.
	readsOneChecks = `[{"events":2},{"events":4}]`
	// writePasses is the result of the check of the write: a pass in one step,
	// which calls Next once.
	writePasses = `{"expect":"pass","detail":{"steps":1},"next-calls":1}`
	// readPasses is the result of the check of the read: a pass in two steps,
	// which calls Next once for the read.
	readPasses = `{"expect":"pass","detail":{"steps":2},"next-calls":1}`
)

// TestResume drives the rules of the resume runner that the definition's
// vectors cannot. Written with testing rather than with this library,
// because a verdict is not written with the subject.
func TestResume(t *testing.T) {
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
				name: "returns nil for the checks of a history whose second check continues the search",
				give: resumed("register", readsOne, readsOneChecks, `[`+writePasses+`,`+readPasses+`]`),
			},
			{
				name: "returns nil for a check that reads the events of the check before without a call of Next",
				give: resumed("register", readsOne, `[{"events":2},{"events":2}]`,
					`[`+writePasses+`,{"expect":"pass","detail":{"steps":1},"next-calls":0}]`),
			},
			{
				name:       "returns a fault at the spec of a vector that names no spec",
				give:       resumed("widget", readsOne, readsOneChecks, `[`+writePasses+`,`+readPasses+`]`),
				wantPath:   inVector(fault.Field(specAt)),
				wantReason: `"widget" is no named spec`,
			},
			{
				name:       "returns a fault at the results of a vector that states fewer results than checks",
				give:       resumed("register", readsOne, readsOneChecks, `[`+writePasses+`]`),
				wantPath:   inVector(fault.Field(resultsAt)),
				wantReason: "the vector states 1 results, want one for each of 2 checks",
			},
			{
				name:       "returns a fault at the events of a check that states none",
				give:       resumed("register", readsOne, `[{}]`, `[`+writePasses+`]`),
				wantPath:   inVector(fault.Field(checksAt), fault.Index(0), fault.Field(eventsAt)),
				wantReason: "the check reads null events, want 1 to 4",
			},
			{
				name:       "returns a fault at the events of a check past the end of the history",
				give:       resumed("register", readsOne, `[{"events":5}]`, `[`+writePasses+`]`),
				wantPath:   inVector(fault.Field(checksAt), fault.Index(0), fault.Field(eventsAt)),
				wantReason: "the check reads 5 events, want 1 to 4",
			},
			{
				name: "returns a fault at the events of a check that reads fewer events than the check before",
				give: resumed("register", readsOne, `[{"events":4},{"events":2}]`,
					`[{"expect":"pass","detail":{"steps":2},"next-calls":2},`+writePasses+`]`),
				wantPath:   inVector(fault.Field(checksAt), fault.Index(1), fault.Field(eventsAt)),
				wantReason: "the check reads 2 events, want 4 to 4",
			},
			{
				name:       "returns a fault at a budget below 1",
				give:       resumed("register", readsOne, `[{"events":2,"budget":0}]`, `[`+writePasses+`]`),
				wantPath:   inVector(fault.Field(checksAt), fault.Index(0), fault.Field(budgetAt)),
				wantReason: "the budget is 0, below 1",
			},
			{
				name: "returns a fault at an entry that is no JSON object among the events of a later check",
				give: resumed("register", `[`+invoking(0, 0)+`,{"ok":0,"output":{"type":"null"}},`+
					invoking(1, 1)+`,[]]`, `[{"events":2},{"events":4}]`, `[`+writePasses+`,`+readPasses+`]`),
				wantPath:   inVector(fault.Field(historyAt), fault.Index(3)),
				wantReason: notParsed,
			},
			{
				name: "returns a fault at an entry of the history that the seam refuses",
				give: resumed("register", `[`+invoking(0, 0)+`,`+invoking(1, 0)+`]`, `[{"events":2}]`,
					`[`+writePasses+`]`),
				wantPath:   inVector(fault.Field(historyAt), fault.Index(1)),
				wantReason: "the seam refuses the entry",
			},
			{
				name:       "returns a fault at the expectation of a failure for a check that passes",
				give:       resumed("register", readsOne, `[{"events":2}]`, `[{"expect":"fail","detail":{}}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(expectAt)),
				wantReason: "the check ends as pass, want fail",
			},
			{
				name:       "returns a fault at the expectation of a pass for a check that fails",
				give:       resumed("register", readsNull, `[{"events":4}]`, `[`+readPasses+`]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(expectAt)),
				wantReason: "the check ends as fail, want pass",
			},
			{
				name:       "returns a fault at an expectation of neither pass nor fail",
				give:       resumed("register", readsOne, `[{"events":2}]`, `[{"expect":"maybe"}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(expectAt)),
				wantReason: `the vector expects "maybe", neither pass nor fail`,
			},
			{
				name: "returns a fault at a field of the detail of a failure that the record states differently",
				give: resumed("register", readsNull, `[{"events":4}]`,
					`[{"expect":"fail","detail":{"steps":3},"next-calls":2}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(detailAt), fault.Key(stepsAt)),
				wantReason: "the field is 2, want 3",
			},
			{
				name: "returns a fault at the steps of a pass that a check of every call takes in other steps",
				give: resumed("register", readsOne, `[{"events":2}]`,
					`[{"expect":"pass","detail":{"steps":2},"next-calls":1}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(detailAt), fault.Key(stepsAt)),
				wantReason: "a check of every call takes 1 steps, want 2",
			},
			{
				name: "returns a fault at the steps of a pass that are no number",
				give: resumed("register", readsOne, `[{"events":2}]`,
					`[{"expect":"pass","detail":{"steps":"one"},"next-calls":1}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(detailAt), fault.Key(stepsAt)),
				wantReason: `the detail states the steps "one", want a number`,
			},
			{
				name: "returns a fault at the calls of Next of a check that continues the search",
				give: resumed("register", readsOne, readsOneChecks,
					`[`+writePasses+`,{"expect":"pass","detail":{"steps":2},"next-calls":2}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(1), fault.Field(nextCallsAt)),
				wantReason: "the check calls Next 1 times, want 2",
			},
			{
				name: "returns a fault at the calls of Next of a check that fails",
				give: resumed("register", readsNull, `[{"events":4}]`,
					`[{"expect":"fail","detail":{"steps":2},"next-calls":3}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(nextCallsAt)),
				wantReason: "the check calls Next 2 times, want 3",
			},
			{
				name: "returns a fault at the calls of Next of a result that states none",
				give: resumed("register", readsOne, `[{"events":2}]`,
					`[{"expect":"pass","detail":{"steps":1}}]`),
				wantPath:   inVector(fault.Field(resultsAt), fault.Index(0), fault.Field(nextCallsAt)),
				wantReason: "the check calls Next 1 times, want null",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Resume, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// resumed returns a resume vector of the named spec, the script, the checks
// and the results, each of the last three a JSON text.
func resumed(spec, script, checks, results string) string {
	return fmt.Sprintf(`{"spec":%q,"history":%s,"checks":%s,"results":%s}`, spec, script, checks, results)
}
