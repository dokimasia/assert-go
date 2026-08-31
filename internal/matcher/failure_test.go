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
