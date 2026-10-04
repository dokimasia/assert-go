// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// boxed keeps its contents unexported, which is what most real types
// do and what cmp refuses to walk without an exporter.
type boxed struct{ items []int }

// explodes has an Equal method that panics, so cmp cannot compute a diff
// of it under any options.
type explodes struct{}

// Equal panics, as the method of a value that cmp cannot compare does.
func (explodes) Equal(explodes) bool { panic("this type cannot be compared") }

// selfContaining returns a map whose one entry is the map itself.
func selfContaining() map[string]any {
	m := map[string]any{}
	m["self"] = m
	return m
}

// sentenceID is the assertion whose sentence this test binary registers.
const sentenceID = "matcher-test-sentence"

// The registration runs while the test binary initialises, as the
// registration of a package does.
func init() {
	matcher.RegisterSentence(func(f matcher.Failure) string { return "registered: " + f.Contract }, sentenceID)
}

// TestWriter checks the sentence that the text writer writes for a
// failure's record, the only text that these tests compare besides the
// text of a fault.
func TestWriter(t *testing.T) {
	t.Parallel()

	t.Run("Render", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give matcher.Failure
			want string
		}{
			{
				name: "returns the contract alone for a record without detail",
				give: matcher.Failure{Assertion: "true", Contract: "the flag is set"},
				want: "the flag is set",
			},
			{
				name: "names want before got",
				give: matcher.Failure{
					Assertion: "length", Contract: "every item comes back",
					Detail: map[string]any{"got": 2, "want": 3},
				},
				want: "every item comes back: want 3, got 2",
			},
			{
				name: "sorts a field it does not know after the ones it does",
				give: matcher.Failure{
					Assertion: "made-up", Contract: "the contract",
					Detail: map[string]any{"zebra": 1, "got": 2, "apple": 3},
				},
				want: "the contract: got 2, apple 3, zebra 1",
			},
			{
				name: "lists the fields of an equality record without got",
				give: matcher.Failure{
					Assertion: "equal",
					Contract:  "the count is right",
					Detail:    map[string]any{"want": 2},
				},
				want: "the count is right: want 2",
			},
			{
				name: "writes a value that contains itself with the cycle marked",
				give: matcher.Failure{
					Assertion: "contains",
					Contract:  "the graph has the node",
					Detail:    map[string]any{"haystack": selfContaining(), "needle": 1},
				},
				want: "the graph has the node: haystack map[self:<cycle>], needle 1",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				if got := matcher.Render(tt.give); got != tt.want {
					t.Fatalf("Render() = %q, want %q", got, tt.want)
				}
			})
		}

		for _, id := range []string{"equal", "golden-match", "golden-match-at", "golden-match-json-field"} {
			t.Run("shows a diff for a mismatch of "+id, func(t *testing.T) {
				t.Parallel()

				got := matcher.Render(matcher.Failure{
					Assertion: id,
					Contract:  "the output matches",
					Detail:    map[string]any{"want": "recorded", "got": "current"},
				})
				if !strings.HasPrefix(got, "the output matches: (-want +got)\n") {
					t.Fatalf("Render() = %q, want a diff", got)
				}
			})
		}

		t.Run("renders a diff of a value with unexported fields", func(t *testing.T) {
			t.Parallel()

			out := matcher.Render(matcher.Failure{
				Assertion: "equal",
				Contract:  "the boxes match",
				Detail:    map[string]any{"want": boxed{items: []int{2}}, "got": boxed{items: []int{1}}},
			})
			if !strings.Contains(out, "-want +got") || !strings.Contains(out, "items") {
				t.Errorf("Render() = %q, want a diff that names the unexported field", out)
			}
		})

		t.Run("renders the values of a value that cmp cannot compare", func(t *testing.T) {
			t.Parallel()

			out := matcher.Render(matcher.Failure{
				Assertion: "equal",
				Contract:  "the values match",
				Detail:    map[string]any{"want": explodes{}, "got": explodes{}},
			})
			if want := "the values match: want {}, got {}"; out != want {
				t.Errorf("Render() = %q, want %q", out, want)
			}
		})

		t.Run("returns the registered sentence of a record's assertion", func(t *testing.T) {
			t.Parallel()

			got := matcher.Render(matcher.Failure{
				Assertion: sentenceID, Contract: "the run passes", Detail: map[string]any{"outcome": "flaky"},
			})
			if want := "registered: the run passes"; got != want {
				t.Fatalf("Render() = %q, want %q", got, want)
			}
		})
	})

	t.Run("RenderFault", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the text that the fault's Error method writes", func(t *testing.T) {
			t.Parallel()

			err := fault.In("prop.ForAll", fault.At(fault.New("the key states no value"), fault.Field("min")))
			if got := matcher.RenderFault(err); got != err.Error() {
				t.Fatalf("RenderFault() = %q, want %q", got, err.Error())
			}
		})
	})

	t.Run("RegisterSentence", func(t *testing.T) {
		t.Parallel()

		t.Run("panics for an assertion whose sentence is registered already and registers nothing", func(t *testing.T) {
			t.Parallel()

			raised := matchertest.Raised(func() {
				matcher.RegisterSentence(func(matcher.Failure) string { return "replaced" }, "true", sentenceID)
			})
			if message, _ := raised.(string); !strings.HasPrefix(message, "matcher: RegisterSentence") {
				t.Fatalf("recovered %v, want the panic of a second registration", raised)
			}
			unregistered := matcher.Failure{Assertion: "true", Contract: "the flag is set"}
			if got := matcher.Render(unregistered); got != "the flag is set" {
				t.Fatalf("Render() = %q after the panic, want the contract alone", got)
			}
		})
	})
}

// rendered keeps the text that a call of the writer returns.
var rendered string

// TestWriterAllocs checks the allocation ceiling of Render and of
// RenderFault. RegisterSentence has none: a test process registers the
// sentence of an assertion once.
func TestWriterAllocs(t *testing.T) {
	checkAllocs(t, writerCases())
}

// BenchmarkWriter measures Render and RenderFault.
func BenchmarkWriter(b *testing.B) {
	benchAllocs(b, writerCases())
}

// writerCases returns a call of Render on the record of a failure of two
// ints and of RenderFault on a fault of an operation and a field, with its
// allocation ceiling, measured.
func writerCases() []allocCase {
	f := matcher.Failure{
		Assertion: "length", Contract: "every item comes back", Detail: map[string]any{"got": 2, "want": 3},
	}
	err := fault.In("prop.ForAll", fault.At(fault.New("the key states no value"), fault.Field("min")))
	return []allocCase{
		{name: "Render", call: func(matcher.Seat) { rendered = matcher.Render(f) }, allocs: 5},
		{name: "RenderFault", call: func(matcher.Seat) { rendered = matcher.RenderFault(err) }, allocs: 2},
	}
}
