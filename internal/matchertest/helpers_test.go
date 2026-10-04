// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"reflect"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/equality"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// inChild skips t unless the process is the child that runs t's broken
// twins. A suite that fails its twins fails the child, which the parent
// requires.
func inChild(t *testing.T) {
	t.Helper()
	if !childtest.InChild(t) {
		t.Skip("runs only in the child process of the test that drives the broken twins")
	}
}

// expectBroken runs the test named child in a child process, and requires
// that the child fails with each of failures in its output.
func expectBroken(t *testing.T, child string, failures ...string) {
	t.Helper()

	out, err := childtest.Run(t, child)
	if err == nil {
		t.Fatalf("the child passed, want each broken twin to fail:\n%s", out)
	}
	for _, failure := range failures {
		if !strings.Contains(out, failure) {
			t.Errorf("the child reported no %q:\n%s", failure, out)
		}
	}
}

// checkTable fails t for a case table that breaks the shape every runner
// assumes: a name for each case, no name twice, the arguments of the
// arity, and at least one passing and one failing case. A suite over a
// table of another shape checks less than its cases state.
func checkTable(t *testing.T, name string, cases []matchertest.Case, arity int) {
	t.Helper()

	if len(cases) == 0 {
		t.Fatalf("%s is empty; its suite would pass having checked nothing", name)
	}

	seen := map[string]bool{}
	passing, failing := 0, 0

	for _, tc := range cases {
		switch {
		case tc.Name == "":
			t.Errorf("%s has a case with no name", name)
		case seen[tc.Name]:
			t.Errorf("%s repeats the case name %q", name, tc.Name)
		}
		seen[tc.Name] = true

		if got := len(tc.Args); got < arity {
			t.Errorf("%s case %q has %d args, want at least %d", name, tc.Name, got, arity)
		}

		if tc.Fails {
			failing++
			continue
		}

		passing++
		if len(tc.Detail) > 0 || tc.Assertion != "" {
			t.Errorf("%s case %q passes but states the record of a failure", name, tc.Name)
		}
	}

	if passing == 0 {
		t.Errorf("%s has no passing case; it would not notice an assertion that always fails", name)
	}
	if failing == 0 {
		t.Errorf("%s has no failing case; it would not notice an assertion that never fails", name)
	}
}

// report records a failure of assertion with detail on s.
func report(s *matchertest.Seat, assertion, msg string, detail map[string]any) {
	s.Report(matcher.Failure{Assertion: assertion, Contract: msg, Detail: detail}, true)
}

// same reports whether x and y are equal under opts, as the assertions of
// internal/matcher compare them.
func same(x, y any, opts []matcher.Option) bool {
	var r equality.Rules
	for _, opt := range opts {
		r = opt(r)
	}
	return equality.Equal(reflect.ValueOf(x), reflect.ValueOf(y), r)
}
