// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestFailure(t *testing.T) {
	t.Parallel()

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the contract alone for a record without detail", func(t *testing.T) {
			t.Parallel()

			got := matcher.Render(matcher.Failure{
				Assertion: "true",
				Contract:  "the flag is set",
			})
			if want := "the flag is set"; got != want {
				t.Fatalf("Render() = %q, want %q", got, want)
			}
		})

		t.Run("names want before got", func(t *testing.T) {
			t.Parallel()

			got := matcher.Render(matcher.Failure{
				Assertion: "length",
				Contract:  "every item comes back",
				Detail:    map[string]any{"got": 2, "want": 3},
			})
			if want := "every item comes back: want 3, got 2"; got != want {
				t.Fatalf("Render() = %q, want %q", got, want)
			}
		})

		t.Run("sorts a field it does not know after the ones it does", func(t *testing.T) {
			t.Parallel()

			got := matcher.Render(matcher.Failure{
				Assertion: "made-up",
				Contract:  "the contract",
				Detail:    map[string]any{"zebra": 1, "got": 2, "apple": 3},
			})
			if want := "the contract: got 2, apple 3, zebra 1"; got != want {
				t.Fatalf("Render() = %q, want %q", got, want)
			}
		})

		t.Run("shows a diff for an equality mismatch", func(t *testing.T) {
			t.Parallel()

			got := matcher.Render(matcher.Failure{
				Assertion: "equal",
				Contract:  "the count is right",
				Detail:    map[string]any{"want": 2, "got": 1},
			})
			if !strings.Contains(got, "(-want +got)") {
				t.Fatalf("Render() = %q, want a diff", got)
			}
		})

		t.Run("lists the fields of an equality record without got", func(t *testing.T) {
			t.Parallel()

			got := matcher.Render(matcher.Failure{
				Assertion: "equal",
				Contract:  "the count is right",
				Detail:    map[string]any{"want": 2},
			})
			if want := "the count is right: want 2"; got != want {
				t.Fatalf("Render() = %q, want %q", got, want)
			}
		})

		for _, id := range []string{"golden-match", "golden-match-at", "golden-match-json-field"} {
			t.Run("shows a diff for a mismatch of "+id, func(t *testing.T) {
				t.Parallel()

				got := matcher.Render(matcher.Failure{
					Assertion: id,
					Contract:  "the output matches the golden file",
					Detail:    map[string]any{"want": "recorded", "got": "current"},
				})
				if !strings.Contains(got, "(-want +got)") {
					t.Fatalf("Render() = %q, want a diff", got)
				}
			})
		}
	})

	t.Run("Fail", func(t *testing.T) {
		t.Parallel()

		t.Run("reports a record of the aborting mode through Fatalf", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Fail(seat, matcher.Fatal, "equal", "the values match", nil)
			if fatals, errs := len(seat.Fatals()), len(seat.Errs()); fatals != 1 || errs != 0 {
				t.Fatalf("reported %d through Fatalf and %d through Errorf, want 1 and 0", fatals, errs)
			}
		})

		t.Run("reports a record of the recording mode through Errorf", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			matcher.Fail(seat, matcher.Soft, "equal", "the values match", nil)
			if fatals, errs := len(seat.Fatals()), len(seat.Errs()); fatals != 0 || errs != 1 {
				t.Fatalf("reported %d through Fatalf and %d through Errorf, want 0 and 1", fatals, errs)
			}
		})

		t.Run("names the line of the test that called the assertion", func(t *testing.T) {
			t.Parallel()

			seat := &matchertest.Seat{}
			_, file, line, _ := runtime.Caller(0)
			matcher.Equal(seat, matcher.Fatal, 1, 2, "the values match")

			records := seat.Records()
			if len(records) != 1 {
				t.Fatalf("reported %d records, want 1", len(records))
			}
			if want := (matcher.Where{File: file, Line: line + 1}); records[0].Where != want {
				t.Fatalf("the record names %+v, want %+v", records[0].Where, want)
			}
		})
	})

	t.Run("CallerWhere", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the zero Where for frames of the runtime alone", func(t *testing.T) {
			t.Parallel()

			var pcs [1]uintptr
			n := runtime.Callers(0, pcs[:])
			if got := matcher.CallerWhere(pcs[:n]); got != (matcher.Where{}) {
				t.Fatalf("CallerWhere() = %+v, want the zero Where", got)
			}
		})

		t.Run("returns a frame of a package outside this module", func(t *testing.T) {
			t.Parallel()

			var got matcher.Where
			slices.SortFunc([]int{2, 1}, func(a, b int) int {
				var pcs [8]uintptr
				n := runtime.Callers(2, pcs[:])
				got = matcher.CallerWhere(pcs[:n])
				return a - b
			})
			if !strings.Contains(got.File, "/slices/") {
				t.Fatalf("CallerWhere() = %+v, want a file of the slices package", got)
			}
		})
	})
}

// boxed keeps its contents unexported, which is what most real types
// do and what cmp refuses to walk without an exporter.
type boxed struct{ items []int }

// explodes has an Equal method that panics, so cmp cannot compute a diff
// of it under any options.
type explodes struct{}

// Equal panics, as the method of a value that cmp cannot compare does.
func (explodes) Equal(explodes) bool { panic("this type cannot be compared") }

// TestRenderOfValuesThatCmpCannotWalk pins that rendering a failure never
// panics. cmp panics on an unexported field when it has no exporter, and
// on a value whose Equal method panics.
func TestRenderOfValuesThatCmpCannotWalk(t *testing.T) {
	t.Parallel()

	t.Run("a value with unexported fields renders a diff", func(t *testing.T) {
		t.Parallel()

		out := matcher.Render(matcher.Failure{
			Assertion: "equal",
			Contract:  "the boxes match",
			Detail: map[string]any{
				"want": boxed{items: []int{2}},
				"got":  boxed{items: []int{1}},
			},
		})

		if !strings.Contains(out, "-want +got") {
			t.Errorf("Render() = %q, want a diff", out)
		}
		if !strings.Contains(out, "items") {
			t.Errorf("Render() = %q, want the unexported field named", out)
		}
	})

	t.Run("a value that cmp cannot compare renders the values", func(t *testing.T) {
		t.Parallel()

		out := matcher.Render(matcher.Failure{
			Assertion: "equal",
			Contract:  "the values match",
			Detail:    map[string]any{"want": explodes{}, "got": explodes{}},
		})

		if strings.Contains(out, "-want +got") {
			t.Errorf("Render() = %q, want the values rather than a diff", out)
		}
		if !strings.Contains(out, "want") || !strings.Contains(out, "got") {
			t.Errorf("Render() = %q, want both values named", out)
		}
	})
}
