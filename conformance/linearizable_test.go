// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The members of a linearizable vector that the paths of the tests name.
const (
	specAt      = "spec"
	budgetAt    = "budget"
	memoLimitAt = "memo-limit"
	workersAt   = "workers"
	stepsAt     = "steps"
)

// The scripts of the histories of a linearizable vector that several tests
// build.
const (
	// readsNull is a script in which client 0 writes 1 and client 1 then
	// reads null, which the register rejects after the write.
	readsNull = `[{"invoke":0,"client":0,"operation":"write","args":[{"type":"int","value":1}],` +
		`"keys":[{"type":"string","value":"x"}]},{"ok":0,"output":{"type":"null"}},` +
		`{"invoke":1,"client":1,"operation":"read","args":[],"keys":[{"type":"string","value":"x"}]},` +
		`{"ok":1,"output":{"type":"null"}}]`
	// readsOne is a script in which client 0 writes 1 and client 1 then
	// reads 1, which passes in two steps.
	readsOne = `[{"invoke":0,"client":0,"operation":"write","args":[{"type":"int","value":1}],` +
		`"keys":[{"type":"string","value":"x"}]},{"ok":0,"output":{"type":"null"}},` +
		`{"invoke":1,"client":1,"operation":"read","args":[],"keys":[{"type":"string","value":"x"}]},` +
		`{"ok":1,"output":{"type":"int","value":1}}]`
	// readsTwo is a script in which clients 0 and 1 write 1 and 2 at once,
	// and client 0 then reads 2, which passes in three steps.
	readsTwo = `[{"invoke":0,"client":0,"operation":"write","args":[{"type":"int","value":1}],"keys":[]},` +
		`{"invoke":1,"client":1,"operation":"write","args":[{"type":"int","value":2}],"keys":[]},` +
		`{"ok":0,"output":{"type":"null"}},{"ok":1,"output":{"type":"null"}},` +
		`{"invoke":2,"client":0,"operation":"read","args":[],"keys":[]},{"ok":2,"output":{"type":"int","value":2}}]`
)

// The vectors of readsNull with a memo limit of 1 and on 1 worker, whose
// outputs the reference computes.
const (
	// memoOfOne is the vector of readsNull under a memo limit of 1.
	memoOfOne = `{"spec":"register","history":` + readsNull + `,"memo-limit":1,"expect":"fail",` +
		`"detail":{"outcome":"undecided","partitions":1,"steps":1,"partition":[{"type":"string","value":"x"}],` +
		`"calls":2,"concurrency":1,"linearized":[],"states":[{"type":"null"}],"candidates":[],"limit":"memo"}}`
	// oneWorker is the vector of readsNull on 1 worker.
	oneWorker = `{"spec":"register","history":` + readsNull + `,"workers":1,"expect":"fail",` +
		`"detail":{"outcome":"violated","partitions":1,"steps":2,"partition":[{"type":"string","value":"x"}],` +
		`"calls":2,"concurrency":1,"linearized":[{"call":0,"completion":1,"process":0,"operation":"write",` +
		`"args":[{"type":"int","value":1}],"output":{"type":"null"}}],"states":[{"type":"int","value":1}],` +
		`"candidates":[{"call":2,"completion":3,"process":1,"operation":"read","args":[],` +
		`"output":{"type":"null"}}],"limit":null}}`
)

// noPassSteps is the reason of a passing vector whose steps no record
// states.
const noPassSteps = "the runner observes the steps of a pass over one partition in two steps or more, " +
	"and the vector states %s partitions and %s steps"

// TestLinearizable drives the rules of the linearizable runner that the
// definition's vectors cannot. Written with testing rather than with this
// library, because a verdict is not written with the subject.
func TestLinearizable(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{name: "returns nil for a vector of a memo limit of 1", give: memoOfOne},
			{name: "returns nil for a vector of 1 worker", give: oneWorker},
			{
				name:       "returns a fault at the spec of a vector that names no spec",
				give:       checked("widget", readsNull, "", "fail", `{}`),
				wantPath:   inVector(fault.Field(specAt)),
				wantReason: `"widget" is no named spec`,
			},
			{
				name:       "returns a fault at an entry of the history that states no call of the seam",
				give:       checked("register", `[{}]`, "", "fail", `{}`),
				wantPath:   inVector(fault.Field(historyAt), fault.Index(0)),
				wantReason: "the entry states no call of the seam",
			},
			{
				name:       "returns a fault at an entry of the history that the seam refuses",
				give:       checked("register", `[`+invoking(0, 0)+`,`+invoking(1, 0)+`]`, "", "fail", `{}`),
				wantPath:   inVector(fault.Field(historyAt), fault.Index(1)),
				wantReason: "the seam refuses the entry",
			},
			{
				name:       "returns a fault at a budget below 1",
				give:       checked("register", readsNull, `"budget":0,`, "fail", `{}`),
				wantPath:   inVector(fault.Field(budgetAt)),
				wantReason: "the budget is 0, below 1",
			},
			{
				name:       "returns a fault at a memo limit below 1",
				give:       checked("register", readsNull, `"memo-limit":0,`, "fail", `{}`),
				wantPath:   inVector(fault.Field(memoLimitAt)),
				wantReason: "the memo limit is 0, below 1",
			},
			{
				name:       "returns a fault at workers below 1",
				give:       checked("register", readsNull, `"workers":0,`, "fail", `{}`),
				wantPath:   inVector(fault.Field(workersAt)),
				wantReason: "the workers are 0, below 1",
			},
			{
				name:       "returns a fault at an expectation of neither pass nor fail",
				give:       checked("register", readsNull, "", "maybe", `{}`),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: `the vector expects "maybe", neither pass nor fail`,
			},
			{
				name:       "returns a fault at the expectation of a failure for a check that passes",
				give:       checked("register", readsOne, "", "fail", `{}`),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: "the check ends as pass, want fail",
			},
			{
				name:       "returns a fault at a field of the detail that the record states differently",
				give:       checked("register", readsNull, "", "fail", `{"steps":3}`),
				wantPath:   inVector(fault.Field(detailAt), fault.Key(stepsAt)),
				wantReason: "the field is 2, want 3",
			},
			{
				name:       "returns a fault at a field of the detail that the record does not state",
				give:       checked("register", readsNull, "", "fail", `{"widget":1}`),
				wantPath:   inVector(fault.Field(detailAt), fault.Key("widget")),
				wantReason: "the record states no such field, want 1",
			},
			{
				name:       "returns a fault at the expectation of a pass for a check that fails",
				give:       checked("register", readsNull, "", "pass", `{"partitions":1,"steps":2}`),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: "the check ends as fail, want pass",
			},
			{
				name:       "returns a fault at the detail of a pass over two partitions",
				give:       checked("register", readsOne, "", "pass", `{"partitions":2,"steps":2}`),
				wantPath:   inVector(fault.Field(detailAt)),
				wantReason: fmt.Sprintf(noPassSteps, "2", "2"),
			},
			{
				name:       "returns a fault at the detail of a pass of one step",
				give:       checked("register", readsOne, "", "pass", `{"partitions":1,"steps":1}`),
				wantPath:   inVector(fault.Field(detailAt)),
				wantReason: fmt.Sprintf(noPassSteps, "1", "1"),
			},
			{
				name:       "returns a fault at the detail of a pass whose steps are no number",
				give:       checked("register", readsOne, "", "pass", `{"partitions":1,"steps":"two"}`),
				wantPath:   inVector(fault.Field(detailAt)),
				wantReason: fmt.Sprintf(noPassSteps, "1", `"two"`),
			},
			{
				name:       "returns a fault at the detail of a pass whose partitions are no number",
				give:       checked("register", readsOne, "", "pass", `{"partitions":"one","steps":2}`),
				wantPath:   inVector(fault.Field(detailAt)),
				wantReason: fmt.Sprintf(noPassSteps, `"one"`, "2"),
			},
			{
				name:     "returns a fault at the steps of a pass that passes within one step less",
				give:     checked("register", readsOne, "", "pass", `{"partitions":1,"steps":3}`),
				wantPath: inVector(fault.Field(detailAt), fault.Key(stepsAt)),
				wantReason: `the check within 2 steps ends as pass with {"outcome":"","partitions":0,"steps":0,"limit":null},` +
					` want {"outcome":"undecided","partitions":1,"steps":2,"limit":"steps"}`,
			},
			{
				name:       "returns a fault at the steps of a pass that needs more steps",
				give:       checked("register", readsTwo, "", "pass", `{"partitions":1,"steps":2}`),
				wantPath:   inVector(fault.Field(detailAt), fault.Key(stepsAt)),
				wantReason: "the check ends as fail within 2 steps, want pass",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Linearizable, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// checked returns a linearizable vector of the named spec and the script,
// with the options, a JSON text of members that ends with a comma, the
// expectation and the detail.
func checked(spec, script, options, expect, detail string) string {
	return fmt.Sprintf(`{"spec":%q,"history":%s,%s"expect":%q,"detail":%s}`, spec, script, options, expect, detail)
}
