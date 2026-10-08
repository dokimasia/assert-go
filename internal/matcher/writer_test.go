// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// sentenceID is the assertion whose sentence this test binary registers.
const sentenceID = "matcher-test-sentence"

// The registration runs while the test binary initialises, as the
// registration of a package does.
func init() {
	matcher.RegisterSentence(func(f matcher.Failure) string { return "registered: " + f.Contract }, sentenceID)
}

// boxed keeps its contents in an unexported field.
type boxed struct{ items []int }

// explodes has an Equal method that panics, which no comparison runs.
type explodes struct{}

// Equal panics.
func (explodes) Equal(explodes) bool { panic("this type cannot be compared") }

// order is a value with a slice of structs, a slice of strings and a map.
type order struct {
	Lines []line
	Tags  []string
	Notes map[string]string
}

// line is one line of an order.
type line struct {
	Quantity int
}

// tag is an int whose String method writes a name.
type tag int

// String returns the name of every tag.
func (tag) String() string { return "tag" }

// tagged contains a tag in an unexported field.
type tagged struct {
	t tag
}

// body is a string of another type than string.
type body string

// silent is a value whose String method writes no text.
type silent struct{}

// String returns the empty string.
func (silent) String() string { return "" }

// letters returns the ten lines a to j, with the line of index changed to
// its upper case.
func letters(index int) string {
	lines := strings.Split("a\nb\nc\nd\ne\nf\ng\nh\ni\nj", "\n")
	if index >= 0 {
		lines[index] = strings.ToUpper(lines[index])
	}
	return strings.Join(lines, "\n")
}

// counting returns the ints from start to start + n - 1.
func counting(start, n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = start + i
	}
	return out
}

// selfContaining returns a map whose one entry is the map itself.
func selfContaining() map[string]any {
	m := map[string]any{}
	m["self"] = m
	return m
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
				name: "writes a whole float of a million or more in decimal",
				give: matcher.Failure{
					Assertion: "in-range", Contract: "the allocations per iteration are within their ceiling",
					Detail: map[string]any{"got": 4194298.0, "low": 0.0, "high": 4194000.0},
				},
				want: "the allocations per iteration are within their ceiling: got 4194298, low 0, high 4194000",
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
				name: "lists the fields of an equality record without want",
				give: matcher.Failure{
					Assertion: "equal",
					Contract:  "the count is right",
					Detail:    map[string]any{"got": 2},
				},
				want: "the count is right: got 2",
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

		t.Run("states the fields of two values that differ in no place", func(t *testing.T) {
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

		self := selfContaining()
		diffs := []struct {
			name     string
			giveWant any
			giveGot  any
			want     string
		}{
			{
				name: "writes each place where want and got differ on a line of its own",
				giveWant: order{
					Lines: []line{{Quantity: 1}, {Quantity: 2}, {Quantity: 3}},
					Tags:  []string{"new", "gift"},
					Notes: map[string]string{},
				},
				giveGot: order{
					Lines: []line{{Quantity: 1}, {Quantity: 2}, {Quantity: 4}},
					Tags:  []string{"new"},
					Notes: map[string]string{"courier": "leave at the door"},
				},
				want: "\t.Lines[2].Quantity: -3 +4\n\t.Tags[1]: -\"gift\"\n\t.Notes[\"courier\"]: +\"leave at the door\"\n",
			},
			{name: "writes a place at the roots without a path", giveWant: 1, giveGot: 2, want: "\t-1 +2\n"},
			{
				name:     "writes an int key bare",
				giveWant: map[int]string{7: "a"},
				giveGot:  map[int]string{7: "b"},
				want:     "\t[7]: -\"a\" +\"b\"\n",
			},
			{
				name:     "writes the types of two values whose texts are equal",
				giveWant: 1,
				giveGot:  int64(1),
				want:     "\t-int(1) +int64(1)\n",
			},
			{
				name:     "writes a value that contains itself with the cycle marked",
				giveWant: []any{self},
				giveGot:  []any{1},
				want:     "\t[0]: -map[self:<cycle>] +1\n",
			},
			{
				name:     "writes a place that want alone has, whose text is empty",
				giveWant: []any{silent{}},
				giveGot:  []any{},
				want:     "\t[0]: -\n",
			},
			{
				name:     "writes a place that got alone has, whose text is empty",
				giveWant: []any{},
				giveGot:  []any{silent{}},
				want:     "\t[0]: +\n",
			},
			{
				name:     "writes a nil element of an interface type as nil",
				giveWant: []any{nil},
				giveGot:  []any{1},
				want:     "\t[0]: -<nil> +1\n",
			},
			{
				name:     "writes a value of an unexported field without its methods",
				giveWant: tagged{t: 1},
				giveGot:  tagged{t: 2},
				want:     "\t.t: -1 +2\n",
			},
			{
				name:     "writes a line diff of two texts of more than one line, three lines around each change",
				giveWant: letters(-1),
				giveGot:  letters(4),
				want:     "\t  …\n\t  b\n\t  c\n\t  d\n\t- e\n\t+ E\n\t  f\n\t  g\n\t  h\n\t  …\n",
			},
			{
				name:     "writes a line diff without a gap at a change near both ends",
				giveWant: "a\nb",
				giveGot:  "a\nc",
				want:     "\t  a\n\t- b\n\t+ c\n",
			},
			{
				name:     "writes a line diff of a text of one line against a text of more",
				giveWant: "a",
				giveGot:  "a\nb",
				want:     "\t  a\n\t+ b\n",
			},
			{
				name:     "writes a line diff of a text of more than one line against a text of one",
				giveWant: "a\nb",
				giveGot:  "a",
				want:     "\t  a\n\t- b\n",
			},
			{
				name:     "writes the path of a line diff on a line of its own",
				giveWant: struct{ Body string }{Body: "x\ny"},
				giveGot:  struct{ Body string }{Body: "x\nz"},
				want:     "\t.Body:\n\t  x\n\t- y\n\t+ z\n",
			},
			{
				name:     "writes two texts of other types quoted",
				giveWant: "a\nb",
				giveGot:  body("a\nc"),
				want:     "\t-\"a\\nb\" +\"a\\nc\"\n",
			},
		}
		for _, tt := range diffs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				got := matcher.Render(matcher.Failure{
					Assertion: "equal",
					Contract:  "the order is stored",
					Detail:    map[string]any{"want": tt.giveWant, "got": tt.giveGot},
				})
				if want := "the order is stored: (-want +got)\n" + tt.want; got != want {
					t.Fatalf("Render() = %q, want %q", got, want)
				}
			})
		}

		t.Run("writes … in place of the places past 64", func(t *testing.T) {
			t.Parallel()

			got := matcher.Render(matcher.Failure{
				Assertion: "equal",
				Contract:  "the counts match",
				Detail:    map[string]any{"want": counting(0, 70), "got": counting(100, 70)},
			})
			lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
			if len(lines) != 66 || lines[64] != "\t[63]: -63 +163" || lines[65] != "\t…" {
				t.Fatalf("Render() = %q, want the contract, 64 places and a mark", got)
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
