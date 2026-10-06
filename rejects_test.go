// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// rejection keeps the records that a call of Rejects returns.
var rejection []assert.Failure

// TestRejects runs the shared cases of the assertion that a check fails,
// and the cases of the aborting surface.
func TestRejects(t *testing.T) {
	t.Parallel()

	t.Run("Rejects", func(t *testing.T) {
		t.Parallel()
		matchertest.RunRejects(t, func(s *matchertest.Seat, msg string, check func(matcher.Seat)) []matcher.Failure {
			return assert.Rejects(s, msg, func(tb assert.TB) { check(tb) })
		})

		t.Run("returns the records of a check that states both surfaces, in call order", func(t *testing.T) {
			t.Parallel()

			got := assert.Rejects(t, "rejects 2 and 3", func(tb assert.TB) {
				expect.Equal(tb, 2, 1, "the first value is one")
				assert.Equal(tb, 3, 1, "the second value is one")
			})

			if len(got) != 2 || got[0].Assertion != "equal" || got[0].Contract != "the first value is one" ||
				got[1].Contract != "the second value is one" {
				t.Fatalf("returned %+v, want the records of both failures in call order", got)
			}
			if got[1].Detail["got"] != 3 || got[1].Detail["want"] != 1 {
				t.Fatalf("the second record states %v, want got 3 and want 1", got[1].Detail)
			}
		})

		t.Run("reports a record of rejects through Fatalf when the driven check passes", func(t *testing.T) {
			t.Parallel()

			outer := &matchertest.Seat{}
			got := assert.Rejects(outer, "rejects 1", func(tb assert.TB) {
				assert.Equal(tb, 1, 1, "the value is one")
			})

			if len(got) != 0 {
				t.Fatalf("returned %+v, want no record of a check that passed", got)
			}
			if len(outer.Fatals()) != 1 || len(outer.Errs()) != 0 {
				t.Fatalf("reported %q through Fatalf and %q through Errorf, want one failure that stops the test",
					outer.Fatals(), outer.Errs())
			}
			records := outer.Records()
			if len(records) != 1 || records[0].Assertion != "rejects" || records[0].Contract != "rejects 1" {
				t.Fatalf("reported %+v, want one record of rejects whose contract is the message", records)
			}
			if len(records[0].Detail) != 0 {
				t.Fatalf("the record states %v, and rejects declares no detail field", records[0].Detail)
			}
		})

		t.Run("records the calls of the check under its own call", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Rejects(r, "the check fails", func(tb assert.TB) {
				assert.Equal(tb, 2, 1, "the value is one")
			})

			got := decoded(t, r.Records())
			if len(got) != 2 || got[0]["assertion"] != "rejects" || got[0]["verdict"] != "pass" ||
				got[0]["seq"] != 1.0 {
				t.Fatalf("Records() = %v, want a pass of rejects as 1 and the check's call", got)
			}
			if got[1]["assertion"] != "equal" || got[1]["verdict"] != "fail" || got[1]["parent"] != 1.0 ||
				got[1]["run"] != 1.0 {
				t.Fatalf("the second record is %v, want the check's failure under 1 in run 1", got[1])
			}
		})

		t.Run("records a failure of rejects when the driven check passes", func(t *testing.T) {
			t.Parallel()

			r := assert.NewRecorder()
			assert.Rejects(r, "the check fails", func(tb assert.TB) {
				assert.Equal(tb, 1, 1, "the value is one")
			})

			got := decoded(t, r.Records())
			if len(got) != 2 || got[0]["assertion"] != "rejects" || got[0]["verdict"] != "fail" {
				t.Fatalf("Records() = %v, want a failure of rejects first", got)
			}
			if got[1]["verdict"] != "pass" || got[1]["parent"] != 1.0 {
				t.Fatalf("the second record is %v, want the check's pass under 1", got[1])
			}
		})

		t.Run("leaves no goroutine behind", func(t *testing.T) {
			t.Parallel()

			// Rejects runs its body on a goroutine of its own, which Goexit
			// ends, and returns after that goroutine has ended.
			done := make(chan struct{})
			go func() {
				defer close(done)
				assert.Rejects(t, "rejects 2", func(tb assert.TB) {
					assert.Equal(tb, 2, 1, "the value is one")
				})
			}()
			<-done
		})
	})
}

// TestRejectsAllocs checks the allocation ceiling of a passing call of
// Rejects.
func TestRejectsAllocs(t *testing.T) {
	alloctest.Check(t, rejectsCases())
}

// BenchmarkRejects measures a passing call of Rejects.
func BenchmarkRejects(b *testing.B) {
	for _, c := range rejectsCases() {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}

// rejectsCases returns a passing call of Rejects of a check that fails at
// a call of True, with its allocation ceiling, measured.
func rejectsCases() []alloctest.Case {
	check := func(tb assert.TB) { assert.True(tb, false, "the check fails") }
	return []alloctest.Case{
		{Name: "Rejects", Call: func(tb assert.TB) { rejection = assert.Rejects(tb, allocContract, check) }, Allocs: 8},
	}
}
