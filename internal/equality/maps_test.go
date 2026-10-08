// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"go.dokimi.dev/assert/internal/equality"
	"go.dokimi.dev/assert/internal/matcher"
)

// The allocation ceilings of the comparisons of maps.
const (
	// mapAllocs is the ceiling of the allocations of Equal of two maps of
	// two string keys: the iterator of one map, and the copies of the keys
	// and the values that reflect reads.
	mapAllocs = 8
	// lookupAllocs is the ceiling of the allocations of HasKey of a string
	// in a map of string keys: the copy of the value that the map index
	// reads.
	lookupAllocs = 2
	// anyMapAllocs is the ceiling of the allocations of Equal of two maps
	// of two interface keys of scalars, which a map index looks up as it
	// looks up strings: the iterators of the walk that checks the keys and
	// of the walk that looks them up, and the copies of the keys and the
	// values that reflect reads.
	anyMapAllocs = 14
)

// pair is a map key of two links, compared field by field.
type pair struct {
	A, B *link
}

// TestMaps checks how two maps match their entries, and how a map finds a
// key that equals a needle.
func TestMaps(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		nan := math.NaN()
		selfMap, otherSelfMap := map[string]any{}, map[string]any{}
		selfMap["self"], otherSelfMap["self"] = selfMap, otherSelfMap
		tests := []struct {
			name      string
			giveX     any
			giveY     any
			giveRules equality.Rules
			want      bool
		}{
			{name: "reports two nil maps equal", giveX: map[string]int(nil), giveY: map[string]int(nil), want: true},
			{name: "reports two empty maps equal", giveX: map[string]int{}, giveY: map[string]int{}, want: true},
			{name: "reports a nil map unequal to an empty one", giveX: map[string]int(nil), giveY: map[string]int{}},
			{
				name:      "reports a nil map equal to an empty one under EquateEmpty",
				giveX:     map[string]int(nil),
				giveY:     map[string]int{},
				giveRules: equality.Rules{EquateEmpty: true},
				want:      true,
			},
			{
				name:      "reports a map with an entry unequal to a nil one under EquateEmpty",
				giveX:     map[string]int{"a": 1},
				giveY:     map[string]int(nil),
				giveRules: equality.Rules{EquateEmpty: true},
			},
			{
				name:      "reports a nil map unequal to a map with an entry under EquateEmpty",
				giveX:     map[string]int(nil),
				giveY:     map[string]int{"a": 1},
				giveRules: equality.Rules{EquateEmpty: true},
			},
			{
				name:  "reports maps of equal entries equal",
				giveX: map[string]int{"a": 1},
				giveY: map[string]int{"a": 1},
				want:  true,
			},
			{
				name:  "reports maps of a different value unequal",
				giveX: map[string]int{"a": 1},
				giveY: map[string]int{"a": 2},
			},
			{
				name:  "reports maps of a different key unequal",
				giveX: map[string]int{"a": 1},
				giveY: map[string]int{"b": 1},
			},
			{
				name:  "reports maps of other lengths unequal",
				giveX: map[string]int{"a": 1, "b": 2},
				giveY: map[string]int{"a": 1},
			},
			{
				name:  "matches pointer keys by their targets",
				giveX: map[*link]int{{ID: 1}: 1, {ID: 2}: 2},
				giveY: map[*link]int{{ID: 2}: 2, {ID: 1}: 1},
				want:  true,
			},
			{
				name:  "matches equal pointer keys by their values",
				giveX: map[*link]int{{ID: 1}: 1, {ID: 1}: 2},
				giveY: map[*link]int{{ID: 1}: 2, {ID: 1}: 1},
				want:  true,
			},
			{
				name:  "reports maps unequal whose equal keys hold other values",
				giveX: map[*link]int{{ID: 1}: 1, {ID: 1}: 2},
				giveY: map[*link]int{{ID: 1}: 1, {ID: 1}: 1},
			},
			{
				name:  "matches each entry of y with one entry of x alone",
				giveX: map[*link]int{{ID: 1}: 1, {ID: 1}: 1},
				giveY: map[*link]int{{ID: 1}: 1, {ID: 1}: 2},
			},
			{
				name:  "reports pointer keys of other targets unequal",
				giveX: map[*link]int{{ID: 1}: 1},
				giveY: map[*link]int{{ID: 2}: 1},
			},
			{
				name:  "reports maps with a NaN key unequal",
				giveX: map[float64]int{nan: 1},
				giveY: map[float64]int{nan: 1},
			},
			{
				name:      "reports maps with a NaN key equal under EquateNaNs",
				giveX:     map[float64]int{nan: 1},
				giveY:     map[float64]int{nan: 1},
				giveRules: equality.Rules{EquateNaNs: true},
				want:      true,
			},
			{
				name:      "reports a NaN key unequal to a number key under EquateNaNs",
				giveX:     map[float64]int{nan: 1},
				giveY:     map[float64]int{1: 1},
				giveRules: equality.Rules{EquateNaNs: true},
			},
			{
				name:  "reports a -0 key equal to a +0 key",
				giveX: map[float64]int{math.Copysign(0, -1): 1},
				giveY: map[float64]int{0: 1},
				want:  true,
			},
			{
				name:  "reports maps with a NaN value unequal",
				giveX: map[string]float64{"a": nan},
				giveY: map[string]float64{"a": nan},
			},
			{
				name:  "looks up interface keys of scalar types",
				giveX: map[any]int{1: 1, "a": 2, nil: 3},
				giveY: map[any]int{"a": 2, 1: 1, nil: 3},
				want:  true,
			},
			{
				name:  "reports interface keys of other types unequal",
				giveX: map[any]int{1: 1},
				giveY: map[any]int{int64(1): 1},
			},
			{
				name:  "matches interface keys that hold pointers by their targets",
				giveX: map[any]int{&link{ID: 1}: 1},
				giveY: map[any]int{&link{ID: 1}: 1},
				want:  true,
			},
			{name: "reports two maps that contain themselves equal", giveX: selfMap, giveY: otherSelfMap, want: true},
			{
				name:  "looks up keys of arrays of ints",
				giveX: map[[2]int]int{{1, 2}: 1},
				giveY: map[[2]int]int{{1, 2}: 1},
				want:  true,
			},
			{
				name:  "looks up keys of structs of ints",
				giveX: map[spot]int{{X: 1, Y: 2}: 1},
				giveY: map[spot]int{{X: 1, Y: 2}: 1},
				want:  true,
			},
			{
				name:  "reports keys of structs of other ints unequal",
				giveX: map[spot]int{{X: 1}: 1},
				giveY: map[spot]int{{X: 2}: 1},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := equality.Equal(reflect.ValueOf(tt.giveX), reflect.ValueOf(tt.giveY), tt.giveRules)
				if got != tt.want {
					t.Fatalf("Equal reports %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("HasKey", func(t *testing.T) {
		t.Parallel()

		nan := math.NaN()
		shared := &link{ID: 1}
		tests := []struct {
			name      string
			giveMap   any
			giveKey   any
			giveRules equality.Rules
			want      bool
		}{
			{name: "finds a key of the map's key type", giveMap: map[string]int{"a": 1}, giveKey: "a", want: true},
			{name: "finds no key that the map lacks", giveMap: map[string]int{"a": 1}, giveKey: "b"},
			{name: "finds no key of another type", giveMap: map[string]int{"a": 1}, giveKey: 1},
			{name: "finds a nil key", giveMap: map[any]int{nil: 1}, giveKey: nil, want: true},
			{name: "finds no nil key in a map without one", giveMap: map[any]int{1: 1}, giveKey: nil},
			{
				name:    "finds no nil key of a map whose keys are no interface",
				giveMap: map[string]int{"a": 1},
				giveKey: nil,
			},
			{
				name:    "finds no key that Go cannot hash, without a panic",
				giveMap: map[any]int{1: 1},
				giveKey: []int{1},
			},
			{name: "finds an interface key of the needle's type", giveMap: map[any]int{1: 1}, giveKey: 1, want: true},
			{name: "finds no interface key of another type", giveMap: map[any]int{1: 1}, giveKey: int64(1)},
			{
				name:    "finds no key of an interface type that the needle does not implement",
				giveMap: map[error]int{errors.New("equality_test: key"): 1},
				giveKey: 1,
			},
			{name: "finds no NaN key", giveMap: map[float64]int{nan: 1}, giveKey: nan},
			{
				name:      "finds a NaN key under EquateNaNs",
				giveMap:   map[float64]int{nan: 1},
				giveKey:   nan,
				giveRules: equality.Rules{EquateNaNs: true},
				want:      true,
			},
			{
				name:    "finds a -0 key for +0",
				giveMap: map[float64]int{math.Copysign(0, -1): 1},
				giveKey: 0.0,
				want:    true,
			},
			{
				name:    "finds a pointer key by its target",
				giveMap: map[*link]int{{ID: 1}: 1},
				giveKey: &link{ID: 1},
				want:    true,
			},
			{
				name:    "finds no pointer key of another target",
				giveMap: map[*link]int{{ID: 1}: 1},
				giveKey: &link{ID: 2},
			},
			{
				name: "finds no key that a failed comparison with another key assumed equal",
				giveMap: map[pair]int{
					{A: shared, B: &link{ID: 2}}: 1,
					{A: shared, B: &link{ID: 2}}: 2,
				},
				giveKey: pair{A: &link{ID: 9}, B: &link{ID: 2}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := equality.HasKey(reflect.ValueOf(tt.giveMap), reflect.ValueOf(tt.giveKey), tt.giveRules)
				if got != tt.want {
					t.Fatalf("HasKey reports %v, want %v", got, tt.want)
				}
			})
		}
	})
}

// TestMapsAllocs checks the allocation ceilings of Equal of two maps of
// string keys and of two maps of interface keys, and of HasKey of a string.
func TestMapsAllocs(t *testing.T) {
	x, y := reflect.ValueOf(map[string]int{"a": 1, "b": 2}), reflect.ValueOf(map[string]int{"a": 1, "b": 2})
	key := reflect.ValueOf("a")
	anyX, anyY := reflect.ValueOf(map[any]int{1: 1, "a": 2}), reflect.ValueOf(map[any]int{1: 1, "a": 2})
	calls := []struct {
		name   string
		call   func()
		allocs float64
	}{
		{
			name:   "Equal of two maps of string keys",
			call:   func() { _ = equality.Equal(x, y, equality.Rules{}) },
			allocs: mapAllocs,
		},
		{
			name:   "HasKey of a string",
			call:   func() { _ = equality.HasKey(x, key, equality.Rules{}) },
			allocs: lookupAllocs,
		},
		{
			name:   "Equal of two maps of interface keys",
			call:   func() { _ = equality.Equal(anyX, anyY, equality.Rules{}) },
			allocs: anyMapAllocs,
		},
	}
	for _, c := range calls {
		got := testing.AllocsPerRun(allocRuns, c.call)
		if matcher.AllocationsCounted() && got > c.allocs {
			t.Errorf("%s allocates %v times, want at most %v", c.name, got, c.allocs)
		}
	}
}

// BenchmarkMaps measures Equal of two maps of string keys and of two maps
// of interface keys, and HasKey of a string.
func BenchmarkMaps(b *testing.B) {
	x, y := reflect.ValueOf(map[string]int{"a": 1, "b": 2}), reflect.ValueOf(map[string]int{"a": 1, "b": 2})

	b.Run("Equal", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Equal(x, y, equality.Rules{})
		}
	})

	b.Run("Equal of interface keys", func(b *testing.B) {
		anyX, anyY := reflect.ValueOf(map[any]int{1: 1, "a": 2}), reflect.ValueOf(map[any]int{1: 1, "a": 2})
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Equal(anyX, anyY, equality.Rules{})
		}
	})

	b.Run("HasKey", func(b *testing.B) {
		key := reflect.ValueOf("a")
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.HasKey(x, key, equality.Rules{})
		}
	})
}
