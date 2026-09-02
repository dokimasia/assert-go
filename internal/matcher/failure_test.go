// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

func TestFailure(t *testing.T) {
	t.Parallel()

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		t.Run("answers the contract alone when nothing is carried", func(t *testing.T) {
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
	})
}

// boxed keeps its contents unexported, which is what most real types
// do and what cmp refuses to walk without an exporter.
type boxed struct{ items []int }

// explodes has an Equal method that panics, so cmp cannot draw a diff
// however it is configured.
type explodes struct{}

// Equal panics, standing in for any value a diff cannot be drawn over.
func (explodes) Equal(explodes) bool { panic("this type cannot be compared") }

// TestRenderDrawsADiffItCannotBeKilledBy pins that rendering a failure
// never takes the test binary with it.
//
// A diff explains a failure rather than deciding one. Both cases here
// reached cmp with no options at all, which panics on the first
// unexported field, so the crash arrived exactly when a test first
// failed.
func TestRenderDrawsADiffItCannotBeKilledBy(t *testing.T) {
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

	t.Run("a value no diff can be drawn over falls back to the values", func(t *testing.T) {
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
