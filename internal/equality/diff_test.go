// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/equality"
	"go.dokimi.dev/assert/internal/matcher"
)

// diffAllocs are the allocations of Diff of two structs that differ in one
// of two fields: a step of the path for each field, the steps of the
// difference's path, and the difference.
const diffAllocs = 4

// manyDifferences is the limit of a Diff that finds every difference of
// the values of a case.
const manyDifferences = 64

// chainLinks is the length of the chains that the depth cases compare,
// longer than the 333 links that a walk of 1,000 levels enters.
const chainLinks = 400

// arrayLink is a node of a list whose next node is in an array.
type arrayLink struct {
	ID   int
	Next [1]*arrayLink
}

// sliceLink is a node of a list whose next node is in a slice.
type sliceLink struct {
	ID   int
	Next []*sliceLink
}

// mapLink is a node of a list whose next node is in a map.
type mapLink struct {
	ID   int
	Next map[string]*mapLink
}

// chains returns, for a list whose next node is in an array, in a slice
// and in a map, a list of chainLinks nodes with the IDs 0 to chainLinks - 1
// and one whose last node has the ID -1.
func chains() map[string][2]any {
	out := map[string][2]any{}
	for _, name := range []string{"array", "slice", "map"} {
		var pair [2]any
		for side, last := range []int{chainLinks - 1, -1} {
			var a *arrayLink
			var s *sliceLink
			var m *mapLink
			for i := chainLinks - 1; i >= 0; i-- {
				id := last
				if i != chainLinks-1 {
					id = i
				}
				a = &arrayLink{ID: id, Next: [1]*arrayLink{a}}
				s = &sliceLink{ID: id, Next: []*sliceLink{s}}
				m = &mapLink{ID: id, Next: map[string]*mapLink{"next": m}}
			}
			pair[side] = map[string]any{"array": a, "slice": s, "map": m}[name]
		}
		out[name] = pair
	}
	return out
}

// differences returns the text of each place where Diff finds x and y to
// differ under r: its path, then the type and the value of x after a -,
// and of y after a +, of each side that has the place. The roots are
// interface values, as a caller that takes its values as any passes them.
func differences(x, y any, r equality.Rules, limit int) []string {
	diffs := equality.Diff(reflect.ValueOf(&x).Elem(), reflect.ValueOf(&y).Elem(), r, limit)
	texts := make([]string, 0, len(diffs))
	for _, d := range diffs {
		var text strings.Builder
		for _, s := range d.Path {
			switch {
			case s.Field != "":
				text.WriteString("." + s.Field)
			case s.Key.IsValid():
				fmt.Fprintf(&text, "[%#v]", s.Key)
			default:
				fmt.Fprintf(&text, "[%d]", s.Index)
			}
		}
		text.WriteString(":")
		if d.X.IsValid() {
			text.WriteString(" -" + side(d.X))
		}
		if d.Y.IsValid() {
			text.WriteString(" +" + side(d.Y))
		}
		texts = append(texts, text.String())
	}
	return texts
}

// side returns the type and the value of v, or of the value inside v for an
// interface that is not nil.
func side(v reflect.Value) string {
	if v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	return fmt.Sprintf("%v(%v)", v.Type(), v)
}

// TestDiff checks the places where Diff finds two values to differ.
func TestDiff(t *testing.T) {
	t.Parallel()

	t.Run("Diff", func(t *testing.T) {
		t.Parallel()

		nan := math.NaN()
		selfLoop := &link{ID: 1}
		selfLoop.Next = selfLoop
		otherLoop := &link{ID: 2}
		otherLoop.Next = otherLoop
		selfList, otherList := []any{nil, 1}, []any{nil, 2}
		selfList[0], otherList[0] = selfList, otherList
		selfMap, otherMap := map[string]any{"n": 1}, map[string]any{"n": 2}
		selfMap["self"], otherMap["self"] = selfMap, otherMap
		nodes := [2]link{{ID: 1}, {ID: 2}}
		first, second := &nodes[0], &nodes[1]
		key := "&equality_test.link{ID:1, Next:(*equality_test.link)(nil)}"
		tests := []struct {
			name      string
			giveX     any
			giveY     any
			giveRules equality.Rules
			giveLimit int
			want      []string
		}{
			{name: "returns no differences for two equal values", giveX: []int{1, 2}, giveY: []int{1, 2}},
			{name: "returns two scalars that differ whole", giveX: 1, giveY: 2, want: []string{": -int(1) +int(2)"}},
			{
				name:  "returns two values of different types whole",
				giveX: 1,
				giveY: int64(1),
				want:  []string{": -int(1) +int64(1)"},
			},
			{
				name:  "returns nil and a value whole",
				giveX: nil,
				giveY: 1,
				want:  []string{": -interface {}(<nil>) +int(1)"},
			},
			{
				name:  "returns each field that differs, in the order of the fields",
				giveX: hidden{1, 2},
				giveY: hidden{3, 4},
				want:  []string{".Shown: -int(1) +int(3)", ".kept: -int(2) +int(4)"},
			},
			{
				name:  "returns a nil interface in a field as a value of its own",
				giveX: holder{V: 1},
				giveY: holder{},
				want:  []string{".V: -int(1) +interface {}(<nil>)"},
			},
			{
				name:  "descends into the targets of two pointers",
				giveX: &link{ID: 1},
				giveY: &link{ID: 2},
				want:  []string{".ID: -int(1) +int(2)"},
			},
			{
				name:  "returns a nil pointer and a pointer whole",
				giveX: (*link)(nil),
				giveY: &link{ID: 1},
				want:  []string{": -*equality_test.link(<nil>) +*equality_test.link(&{1 <nil>})"},
			},
			{
				name:  "returns each element of two arrays that differs",
				giveX: [3]int{1, 2, 3},
				giveY: [3]int{1, 5, 3},
				want:  []string{"[1]: -int(2) +int(5)"},
			},
			{
				name:  "returns an element inserted into a slice at its index in y",
				giveX: []int{1, 2, 3},
				giveY: []int{1, 9, 2, 3},
				want:  []string{"[1]: +int(9)"},
			},
			{
				name:  "returns an element removed from a slice at its index in x",
				giveX: []int{1, 2, 3},
				giveY: []int{1, 3},
				want:  []string{"[1]: -int(2)"},
			},
			{
				name:  "returns a nil element that one slice has alone",
				giveX: []any{nil},
				giveY: []any{},
				want:  []string{"[0]: -interface {}(<nil>)"},
			},
			{
				name:  "pairs a run of replaced elements in order, at their index in x",
				giveX: []int{1, 2, 3, 4},
				giveY: []int{1, 7, 8, 4},
				want:  []string{"[1]: -int(2) +int(7)", "[2]: -int(3) +int(8)"},
			},
			{
				name:  "returns the elements of y that a run leaves without a pair alone",
				giveX: []int{1, 2, 4},
				giveY: []int{1, 7, 8, 9, 4},
				want:  []string{"[1]: -int(2) +int(7)", "[2]: +int(8)", "[3]: +int(9)"},
			},
			{
				name:  "returns the elements of x that a run leaves without a pair alone",
				giveX: []int{1, 2, 3, 4},
				giveY: []int{1, 9, 4},
				want:  []string{"[1]: -int(2) +int(9)", "[2]: -int(3)"},
			},
			{
				name:  "pairs the runs at the start and at the end of two slices apart",
				giveX: []int{1, 2, 3, 4, 5},
				giveY: []int{9, 2, 3, 4, 8},
				want:  []string{"[0]: -int(1) +int(9)", "[4]: -int(5) +int(8)"},
			},
			{
				name:  "descends into a pair of elements",
				giveX: []link{{ID: 1}, {ID: 2}},
				giveY: []link{{ID: 1}, {ID: 3}},
				want:  []string{"[1].ID: -int(2) +int(3)"},
			},
			{
				name:  "returns a nil slice and an empty one whole",
				giveX: []int(nil),
				giveY: []int{},
				want:  []string{": -[]int([]) +[]int([])"},
			},
			{
				name:      "returns no differences for a nil slice and an empty one under EquateEmpty",
				giveX:     []int(nil),
				giveY:     []int{},
				giveRules: equality.Rules{EquateEmpty: true},
			},
			{
				name:  "returns two NaNs as a difference",
				giveX: holder{V: []float64{nan, 1}},
				giveY: holder{V: []float64{nan, 2}},
				want:  []string{".V[0]: -float64(NaN) +float64(NaN)", ".V[1]: -float64(1) +float64(2)"},
			},
			{
				name:      "returns no difference at two NaNs under EquateNaNs",
				giveX:     holder{V: []float64{nan, 1}},
				giveY:     holder{V: []float64{nan, 2}},
				giveRules: equality.Rules{EquateNaNs: true},
				want:      []string{".V[1]: -float64(1) +float64(2)"},
			},
			{
				name:      "returns two slices of equal elements whole under ByIdentity",
				giveX:     holder{V: []int{1}},
				giveY:     holder{V: []int{1}},
				giveRules: equality.Rules{ByIdentity: true},
				want:      []string{".V: -[]int([1]) +[]int([1])"},
			},
			{
				name:  "descends into the values of a key that both maps have",
				giveX: map[string]int{"a": 1, "b": 2},
				giveY: map[string]int{"a": 1, "b": 3},
				want:  []string{`["b"]: -int(2) +int(3)`},
			},
			{
				name:  "returns a key that one map has alone, in the order of the keys",
				giveX: map[string]int{"a": 1, "c": 3},
				giveY: map[string]int{"a": 1, "b": 2, "c": 4},
				want:  []string{`["b"]: +int(2)`, `["c"]: -int(3) +int(4)`},
			},
			{
				name:  "returns a key that only x has",
				giveX: map[string]int{"a": 1},
				giveY: map[string]int{},
				want:  []string{`["a"]: -int(1)`},
			},
			{
				name:  "returns a nil value of a key that one map has alone",
				giveX: map[string]any{},
				giveY: map[string]any{"a": nil},
				want:  []string{`["a"]: +interface {}(<nil>)`},
			},
			{
				name:  "returns a NaN key as removed and added",
				giveX: map[float64]int{nan: 1},
				giveY: map[float64]int{nan: 1},
				want:  []string{"[NaN]: -int(1)", "[NaN]: +int(1)"},
			},
			{
				name:      "pairs a NaN key under EquateNaNs",
				giveX:     map[float64]int{nan: 1},
				giveY:     map[float64]int{nan: 2},
				giveRules: equality.Rules{EquateNaNs: true},
				want:      []string{"[NaN]: -int(1) +int(2)"},
			},
			{
				name:  "pairs pointer keys by their targets",
				giveX: map[*link]int{{ID: 1}: 1},
				giveY: map[*link]int{{ID: 1}: 2},
				want:  []string{"[" + key + "]: -int(1) +int(2)"},
			},
			{
				name:  "pairs an equal entry before an entry of an equal key",
				giveX: map[*link]int{{ID: 1}: 1, {ID: 1}: 2},
				giveY: map[*link]int{{ID: 1}: 2, {ID: 1}: 9},
				want:  []string{"[" + key + "]: -int(1) +int(9)"},
			},
			{
				name:  "orders the keys of an interface type by type, then by value, after nil",
				giveX: map[any]int{"a": 1, 2: 1, 1: 1, nil: 1},
				giveY: map[any]int{"a": 2, 2: 2, 1: 2, nil: 2},
				want: []string{
					"[interface {}(nil)]: -int(1) +int(2)",
					"[1]: -int(1) +int(2)",
					"[2]: -int(1) +int(2)",
					`["a"]: -int(1) +int(2)`,
				},
			},
			{
				name:  "orders bool keys false first",
				giveX: map[bool]int{true: 1, false: 1},
				giveY: map[bool]int{true: 2, false: 2},
				want:  []string{"[false]: -int(1) +int(2)", "[true]: -int(1) +int(2)"},
			},
			{
				name:  "orders unsigned keys by value",
				giveX: map[uint]int{2: 1, 1: 1},
				giveY: map[uint]int{2: 2, 1: 2},
				want:  []string{"[0x1]: -int(1) +int(2)", "[0x2]: -int(1) +int(2)"},
			},
			{
				name:  "orders complex keys by their real parts, then by their imaginary parts",
				giveX: map[complex128]int{complex(1, 2): 1, complex(1, 1): 1, complex(0, 5): 1},
				giveY: map[complex128]int{complex(1, 2): 2, complex(1, 1): 2, complex(0, 5): 2},
				want:  []string{"[(0+5i)]: -int(1) +int(2)", "[(1+1i)]: -int(1) +int(2)", "[(1+2i)]: -int(1) +int(2)"},
			},
			{
				name:  "orders array keys element by element",
				giveX: map[[2]int]int{{1, 2}: 1, {1, 1}: 1},
				giveY: map[[2]int]int{{1, 2}: 2, {1, 1}: 2},
				want:  []string{"[[2]int{1, 1}]: -int(1) +int(2)", "[[2]int{1, 2}]: -int(1) +int(2)"},
			},
			{
				name:  "orders struct keys field by field",
				giveX: map[spot]int{{X: 1, Y: 2}: 1, {X: 1, Y: 1}: 1},
				giveY: map[spot]int{{X: 1, Y: 2}: 2, {X: 1, Y: 1}: 2},
				want: []string{
					"[equality_test.spot{X:1, Y:1}]: -int(1) +int(2)",
					"[equality_test.spot{X:1, Y:2}]: -int(1) +int(2)",
				},
			},
			{
				name:  "orders the entry of x before the entry of y of two keys alike",
				giveX: map[[1]float64]int{{nan}: 1},
				giveY: map[[1]float64]int{{nan}: 1},
				want:  []string{"[[1]float64{NaN}]: -int(1)", "[[1]float64{NaN}]: +int(1)"},
			},
			{
				name:  "orders pointer keys by address",
				giveX: map[*link]int{first: 1, second: 1},
				giveY: map[*link]int{second: 2, first: 2},
				want: []string{
					"[" + key + "]: -int(1) +int(2)",
					"[&equality_test.link{ID:2, Next:(*equality_test.link)(nil)}]: -int(1) +int(2)",
				},
			},
			{
				name:  "returns a nil map and an empty one whole",
				giveX: map[string]int(nil),
				giveY: map[string]int{},
				want:  []string{": -map[string]int(map[]) +map[string]int(map[])"},
			},
			{
				name:  "returns each difference of two lists that contain themselves once",
				giveX: selfLoop,
				giveY: otherLoop,
				want:  []string{".ID: -int(1) +int(2)"},
			},
			{
				name:  "returns each difference of two slices that contain themselves once",
				giveX: selfList,
				giveY: otherList,
				want:  []string{"[1]: -int(1) +int(2)"},
			},
			{
				name:  "returns each difference of two maps that contain themselves once",
				giveX: selfMap,
				giveY: otherMap,
				want:  []string{`["n"]: -int(1) +int(2)`},
			},
			{
				name:      "returns at most limit differences",
				giveX:     []int{},
				giveY:     []int{1, 2, 3},
				giveLimit: 2,
				want:      []string{"[0]: +int(1)", "[1]: +int(2)"},
			},
			{
				name:      "stops its walk at limit differences",
				giveX:     hidden{1, 2},
				giveY:     hidden{3, 4},
				giveLimit: 1,
				want:      []string{".Shown: -int(1) +int(3)"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				limit := tt.giveLimit
				if limit == 0 {
					limit = manyDifferences
				}
				got := differences(tt.giveX, tt.giveY, tt.giveRules, limit)
				if !slices.Equal(got, tt.want) {
					t.Fatalf("Diff returns %q, want %q", got, tt.want)
				}
			})
		}

		t.Run("returns two functions whole", func(t *testing.T) {
			t.Parallel()
			got := equality.Diff(reflect.ValueOf(one), reflect.ValueOf(two), equality.Rules{}, manyDifferences)
			if len(got) != 1 || len(got[0].Path) != 0 {
				t.Fatalf("Diff returns %d differences, want the two functions whole", len(got))
			}
		})

		t.Run("states the two values whole 1,000 levels below the roots", func(t *testing.T) {
			t.Parallel()
			x, y := reflect.ValueOf(list(600, 1)), reflect.ValueOf(list(600, 2))
			got := equality.Diff(x, y, equality.Rules{}, manyDifferences)
			if len(got) != 1 {
				t.Fatalf("Diff returns %d differences, want 1", len(got))
			}
			if steps := len(got[0].Path); steps != 500 {
				t.Fatalf("the difference is %d steps below the roots, want the Next of 500 links", steps)
			}
			if id := got[0].X.Elem().Field(0).Int(); id != 500 {
				t.Fatalf("the difference states the link of ID %d, want 500", id)
			}
		})

		for name, pair := range chains() {
			t.Run("counts a level for each "+name+" of a chain", func(t *testing.T) {
				t.Parallel()
				x, y := reflect.ValueOf(pair[0]), reflect.ValueOf(pair[1])
				got := equality.Diff(x, y, equality.Rules{}, manyDifferences)
				if len(got) != 1 {
					t.Fatalf("Diff returns %d differences, want 1", len(got))
				}
				if steps := len(got[0].Path); steps != 666 {
					t.Fatalf("the difference is %d steps below the roots, want two for each of 333 links", steps)
				}
				if id := got[0].X.Field(0).Int(); id != 333 {
					t.Fatalf("the difference states the link of ID %d, want 333 at 1,000 levels", id)
				}
			})
		}

		t.Run("states -1 as the index of a step to a field or to a key", func(t *testing.T) {
			t.Parallel()
			field := equality.Diff(reflect.ValueOf(hidden{1, 2}), reflect.ValueOf(hidden{1, 3}), equality.Rules{}, 1)
			key := equality.Diff(reflect.ValueOf(map[string]int{"a": 1}), reflect.ValueOf(map[string]int{"a": 2}),
				equality.Rules{}, 1)
			if field[0].Path[0].Index != -1 || key[0].Path[0].Index != -1 {
				t.Fatalf("the steps state the indexes %d and %d, want -1", field[0].Path[0].Index, key[0].Path[0].Index)
			}
		})
	})
}

// TestDiffAllocs checks the allocation ceiling of Diff.
func TestDiffAllocs(t *testing.T) {
	x, y := reflect.ValueOf(hidden{1, 2}), reflect.ValueOf(hidden{1, 3})
	got := testing.AllocsPerRun(allocRuns, func() { _ = equality.Diff(x, y, equality.Rules{}, manyDifferences) })
	if matcher.AllocationsCounted() && got > diffAllocs {
		t.Errorf("Diff of two structs allocates %v times, want at most %v", got, diffAllocs)
	}
}

// BenchmarkDiff measures Diff of two structs that differ in one field.
func BenchmarkDiff(b *testing.B) {
	x, y := reflect.ValueOf(hidden{1, 2}), reflect.ValueOf(hidden{1, 3})
	b.ReportAllocs()
	for b.Loop() {
		_ = equality.Diff(x, y, equality.Rules{}, manyDifferences)
	}
}
