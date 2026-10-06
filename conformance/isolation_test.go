// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// anomalyAt is the member of an isolation vector's detail that the paths of
// the tests name.
const anomalyAt = "anomaly"

// The typed literals of the values of the micro-operations of writeSkew,
// beside oneLiteral.
const (
	// emptyLiteral is the typed literal of the empty list.
	emptyLiteral = `{"type":"list","items":[]}`
	// twoLiteral is the typed literal of the integer 2.
	twoLiteral = `{"type":"int","value":2}`
)

// writeSkew is the script of a write skew, which serializability forbids and
// snapshot isolation permits: each of two transactions reads the key that
// the other appends to, and finds it empty.
var writeSkew = fmt.Sprintf(`[{"invoke":0,"client":0,"operation":"txn","args":[%s,%s],"keys":[%s,%s]},`+
	`{"invoke":1,"client":1,"operation":"txn","args":[%s,%s],"keys":[%s,%s]},`+
	`{"ok":0,"output":{"type":"list","items":[%s,%s]}},{"ok":1,"output":{"type":"list","items":[%s,%s]}}]`,
	microOperation("read", "x", nullLiteral), microOperation("append", "y", oneLiteral), stringOf("x"), stringOf("y"),
	microOperation("read", "y", nullLiteral), microOperation("append", "x", twoLiteral), stringOf("y"), stringOf("x"),
	microOperation("read", "x", emptyLiteral), microOperation("append", "y", oneLiteral),
	microOperation("read", "y", emptyLiteral), microOperation("append", "x", twoLiteral))

// TestIsolation drives the rules of the isolation runners that the
// definition's vectors cannot. Written with testing rather than with this
// library, because a verdict is not written with the subject.
func TestIsolation(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			kind       conformance.VectorKind
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for a write skew that serializable fails with G2",
				kind: conformance.Serializable,
				give: isolated(writeSkew, "fail", `{"anomaly":"G2","kinds":["G2"]}`),
			},
			{
				name: "returns nil for a write skew that snapshot-isolation passes",
				kind: conformance.SnapshotIsolation,
				give: isolated(writeSkew, "pass", `{"anomaly":null}`),
			},
			{
				name:       "returns a fault at an entry of the history that states no call of the seam",
				kind:       conformance.Serializable,
				give:       isolated(`[{}]`, "fail", `{}`),
				wantPath:   inVector(fault.Field(historyAt), fault.Index(0)),
				wantReason: "the entry states no call of the seam",
			},
			{
				name:       "returns a fault at an entry of the history that the seam refuses",
				kind:       conformance.SnapshotIsolation,
				give:       isolated(`[`+invoking(0, 0)+`,`+invoking(1, 0)+`]`, "fail", `{}`),
				wantPath:   inVector(fault.Field(historyAt), fault.Index(1)),
				wantReason: "the seam refuses the entry",
			},
			{
				name:       "returns a fault at an expectation of neither pass nor fail",
				kind:       conformance.Serializable,
				give:       isolated(writeSkew, "maybe", `{}`),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: `the vector expects "maybe", neither pass nor fail`,
			},
			{
				name:       "returns a fault at the expectation of a failure for a check that passes",
				kind:       conformance.SnapshotIsolation,
				give:       isolated(writeSkew, "fail", `{}`),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: "the check ends as pass, want fail",
			},
			{
				name:       "returns a fault at the expectation of a pass for a check that fails",
				kind:       conformance.Serializable,
				give:       isolated(writeSkew, "pass", `{}`),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: "the check ends as fail, want pass",
			},
			{
				name:       "returns a fault at the expectation of a failure for a history of no transactions",
				kind:       conformance.Serializable,
				give:       isolated(`[`+invoking(0, 0)+`]`, "fail", `{}`),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: "the check ends as error, want fail",
			},
			{
				name:       "returns a fault at a field of the detail that the record states differently",
				kind:       conformance.Serializable,
				give:       isolated(writeSkew, "fail", `{"anomaly":"G0"}`),
				wantPath:   inVector(fault.Field(detailAt), fault.Key(anomalyAt)),
				wantReason: `the field is "G2", want "G0"`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, tt.kind, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// isolated returns an isolation vector of the script, with the expectation
// and the detail.
func isolated(script, expect, detail string) string {
	return fmt.Sprintf(`{"history":%s,"expect":%q,"detail":%s}`, script, expect, detail)
}

// microOperation returns the typed literal of the micro-operation of
// function on the string key with value, a typed literal.
func microOperation(function, key, value string) string {
	return fmt.Sprintf(`{"type":"list","items":[%s,%s,%s]}`, stringOf(function), stringOf(key), value)
}
