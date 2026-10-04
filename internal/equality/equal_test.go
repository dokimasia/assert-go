// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

import (
	"math"
	"reflect"
	"runtime/debug"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/equality"
	"go.dokimi.dev/assert/internal/matcher"
)

// The allocations of Equal, measured.
const (
	// scalarAllocs are the allocations of Equal of two ints.
	scalarAllocs = 0
	// sliceAllocs are the allocations of Equal of two slices of ints.
	sliceAllocs = 0
	// structAllocs are the allocations of Equal of two structs of ints.
	structAllocs = 0
	// listAllocs are the allocations of Equal of two lists of three links:
	// the set of the pairs of links that the walk entered, and their log.
	listAllocs = 3
)

// deepLinks is the length of the lists that the depth test compares, far
// more levels than a recursive walk fits in deepStack.
const deepLinks = 100_000

// deepStack is the most bytes of stack that the goroutine of the depth test
// may use.
const deepStack = 1 << 20

// views is a struct of two slices, which can view one array.
type views struct {
	Short, Long []any
}

// always is a type whose Equal method reports every two values equal.
type always struct {
	n int
}

// Equal reports true.
func (always) Equal(always) bool { return true }

// explodes is a type whose Equal method panics.
type explodes struct {
	n int
}

// Equal panics.
func (explodes) Equal(explodes) bool { panic("equality_test: the Equal method ran") }

// one returns 1.
func one() int { return 1 }

// two returns 2.
func two() int { return 2 }

// counter returns a closure of one function literal that returns n.
func counter(n int) func() int { return func() int { return n } }

// TestEqual checks each rule by which Equal compares two values, under
// each relaxation.
func TestEqual(t *testing.T) {
	t.Parallel()

	t.Run("Equal", func(t *testing.T) {
		t.Parallel()

		channel := make(chan int)
		nan := math.NaN()
		selfLoop := &link{ID: 1}
		selfLoop.Next = selfLoop
		otherSelfLoop := &link{ID: 1}
		otherSelfLoop.Next = otherSelfLoop
		twoLoop := &link{ID: 1, Next: &link{ID: 1}}
		twoLoop.Next.Next = twoLoop
		selfList, otherSelfList := []any{nil}, []any{nil}
		selfList[0], otherSelfList[0] = selfList, otherSelfList
		endsInOne, endsInTwo := []any{nil, 1}, []any{nil, 2}
		now := time.Now()
		tests := []struct {
			name      string
			giveX     any
			giveY     any
			giveRules equality.Rules
			want      bool
		}{
			{name: "reports two equal ints equal", giveX: 1, giveY: 1, want: true},
			{name: "reports two different ints unequal", giveX: 1, giveY: 2},
			{name: "reports an int unequal to an int64 of its value", giveX: 1, giveY: int64(1)},
			{name: "reports an int unequal to a float of its value", giveX: 1, giveY: 1.0},
			{name: "reports two equal unsigned integers equal", giveX: uint8(7), giveY: uint8(7), want: true},
			{name: "reports two different unsigned integers unequal", giveX: uint8(7), giveY: uint8(8)},
			{name: "reports two equal bools equal", giveX: true, giveY: true, want: true},
			{name: "reports two different bools unequal", giveX: true, giveY: false},
			{name: "reports two equal strings equal", giveX: "a", giveY: "a", want: true},
			{name: "reports two different strings unequal", giveX: "a", giveY: "b"},
			{name: "reports -0 equal to +0", giveX: math.Copysign(0, -1), giveY: 0.0, want: true},
			{name: "reports two different floats unequal", giveX: 1.5, giveY: 2.5},
			{name: "reports a NaN unequal to a NaN", giveX: nan, giveY: nan},
			{
				name:      "reports a NaN equal to a NaN under EquateNaNs",
				giveX:     nan,
				giveY:     nan,
				giveRules: equality.Rules{EquateNaNs: true},
				want:      true,
			},
			{
				name:      "reports a NaN unequal to a number under EquateNaNs",
				giveX:     nan,
				giveY:     1.0,
				giveRules: equality.Rules{EquateNaNs: true},
			},
			{
				name:      "reports a number unequal to a NaN under EquateNaNs",
				giveX:     1.0,
				giveY:     nan,
				giveRules: equality.Rules{EquateNaNs: true},
			},
			{name: "reports a float32 unequal to a float64 of its value", giveX: float32(1.5), giveY: 1.5},
			{name: "reports two equal complex numbers equal", giveX: complex(1, 2), giveY: complex(1, 2), want: true},
			{name: "reports complex numbers of other real parts unequal", giveX: complex(1, 2), giveY: complex(3, 2)},
			{
				name:  "reports complex numbers of other imaginary parts unequal",
				giveX: complex(1, 2),
				giveY: complex(1, 3),
			},
			{
				name:      "reports complex numbers with NaN parts equal under EquateNaNs",
				giveX:     complex(nan, nan),
				giveY:     complex(nan, nan),
				giveRules: equality.Rules{EquateNaNs: true},
				want:      true,
			},
			{name: "reports a channel equal to itself", giveX: channel, giveY: channel, want: true},
			{name: "reports two channels unequal", giveX: make(chan int), giveY: make(chan int)},
			{name: "reports a function equal to itself", giveX: one, giveY: one, want: true},
			{name: "reports two functions unequal", giveX: one, giveY: two},
			{name: "reports two nil functions equal", giveX: (func() int)(nil), giveY: (func() int)(nil), want: true},
			{name: "reports a nil function unequal to a function", giveX: (func() int)(nil), giveY: one},
			{
				name:  "reports two closures of one literal equal by their code",
				giveX: counter(1),
				giveY: counter(2),
				want:  true,
			},
			{name: "reports two nil pointers equal", giveX: (*int)(nil), giveY: (*int)(nil), want: true},
			{name: "reports a nil pointer unequal to a pointer", giveX: (*int)(nil), giveY: new(0)},
			{name: "reports a pointer unequal to a nil pointer", giveX: new(0), giveY: (*int)(nil)},
			{name: "reports pointers to equal values equal", giveX: new(1), giveY: new(1), want: true},
			{name: "reports pointers to different values unequal", giveX: new(1), giveY: new(2)},
			{name: "reports a pointer to a NaN unequal to itself", giveX: &nan, giveY: &nan},
			{
				name:      "reports a pointer to a NaN equal to itself under EquateNaNs",
				giveX:     &nan,
				giveY:     &nan,
				giveRules: equality.Rules{EquateNaNs: true},
				want:      true,
			},
			{name: "reports nil equal to nil", giveX: nil, giveY: nil, want: true},
			{name: "reports nil unequal to a zero", giveX: nil, giveY: 0},
			{name: "reports a zero unequal to nil", giveX: 0, giveY: nil},
			{name: "reports fields of equal values equal", giveX: holder{V: 1}, giveY: holder{V: 1}, want: true},
			{name: "reports fields of values of other types unequal", giveX: holder{V: 1}, giveY: holder{V: int64(1)}},
			{name: "reports two nil fields equal", giveX: holder{}, giveY: holder{}, want: true},
			{name: "reports a nil field unequal to a zero field", giveX: holder{}, giveY: holder{V: 0}},
			{name: "reports equal arrays equal", giveX: [2]int{1, 2}, giveY: [2]int{1, 2}, want: true},
			{name: "reports arrays of a different element unequal", giveX: [2]int{1, 2}, giveY: [2]int{1, 3}},
			{name: "reports structs of equal fields equal", giveX: hidden{1, 2}, giveY: hidden{1, 2}, want: true},
			{
				name:  "reports structs that differ in an unexported field unequal",
				giveX: hidden{1, 2},
				giveY: hidden{1, 3},
			},
			{
				name:  "reports structs that differ in an exported field unequal",
				giveX: hidden{1, 2},
				giveY: hidden{3, 2},
			},
			{name: "reports two empty structs equal", giveX: struct{}{}, giveY: struct{}{}, want: true},
			{name: "reports two nil slices equal", giveX: []int(nil), giveY: []int(nil), want: true},
			{name: "reports two empty slices equal", giveX: []int{}, giveY: []int{}, want: true},
			{name: "reports a nil slice unequal to an empty one", giveX: []int(nil), giveY: []int{}},
			{
				name:      "reports a nil slice equal to an empty one under EquateEmpty",
				giveX:     []int(nil),
				giveY:     []int{},
				giveRules: equality.Rules{EquateEmpty: true},
				want:      true,
			},
			{
				name:      "reports an empty slice equal to a nil one under EquateEmpty",
				giveX:     []int{},
				giveY:     []int(nil),
				giveRules: equality.Rules{EquateEmpty: true},
				want:      true,
			},
			{
				name:      "reports a slice with an element unequal to a nil one under EquateEmpty",
				giveX:     []int{1},
				giveY:     []int(nil),
				giveRules: equality.Rules{EquateEmpty: true},
			},
			{
				name:      "reports a nil slice unequal to a slice with an element under EquateEmpty",
				giveX:     []int(nil),
				giveY:     []int{1},
				giveRules: equality.Rules{EquateEmpty: true},
			},
			{name: "reports slices of equal elements equal", giveX: []int{1, 2}, giveY: []int{1, 2}, want: true},
			{name: "reports slices of other lengths unequal", giveX: []int{1, 2}, giveY: []int{1}},
			{name: "reports slices of a different last element unequal", giveX: []int{1, 2}, giveY: []int{1, 3}},
			{name: "reports slices of a different first element unequal", giveX: []int{1, 2}, giveY: []int{3, 2}},
			{name: "reports values unequal whose Equal method reports them equal", giveX: always{1}, giveY: always{2}},
			{
				name:  "compares the fields of values whose Equal method panics",
				giveX: explodes{1},
				giveY: explodes{1},
				want:  true,
			},
			{name: "reports a time unequal to itself without its monotonic reading", giveX: now, giveY: now.Round(0)},
			{
				name:  "reports two lists that contain themselves equal",
				giveX: selfLoop,
				giveY: otherSelfLoop,
				want:  true,
			},
			{
				name:  "reports a cycle of one link equal to a cycle of two links alike",
				giveX: selfLoop,
				giveY: twoLoop,
				want:  true,
			},
			{
				name:  "reports a cycle unequal to a list that ends",
				giveX: selfLoop,
				giveY: &link{ID: 1, Next: &link{ID: 1}},
			},
			{
				name:  "reports two slices that contain themselves equal",
				giveX: selfList,
				giveY: otherSelfList,
				want:  true,
			},
			{
				name:  "compares a slice after a shorter slice of its array",
				giveX: views{Short: endsInOne[:1], Long: endsInOne},
				giveY: views{Short: endsInTwo[:1], Long: endsInTwo},
			},
			{name: "reports pointers to equal arrays equal", giveX: &[2]int{1, 2}, giveY: &[2]int{1, 2}, want: true},
			{
				name:  "reports pointers to arrays of pointers to equal values equal",
				giveX: &[1]*int{new(1)},
				giveY: &[1]*int{new(1)},
				want:  true,
			},
			{
				name:  "reports a slice that contains itself unequal to one that contains an empty slice",
				giveX: selfList,
				giveY: []any{[]any{}},
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
}

// TestEqualDepth checks that Equal compares lists far deeper than the
// goroutine's stack would allow a recursive walk. It lowers the maximum
// stack of every goroutine of the process for its run, so it does not run
// in parallel.
func TestEqualDepth(t *testing.T) {
	x, y, other := list(deepLinks, 1), list(deepLinks, 1), list(deepLinks, 2)
	previous := debug.SetMaxStack(deepStack)
	defer debug.SetMaxStack(previous)

	if !equality.Equal(reflect.ValueOf(x), reflect.ValueOf(y), equality.Rules{}) {
		t.Fatal("Equal reports two equal lists unequal")
	}
	if equality.Equal(reflect.ValueOf(x), reflect.ValueOf(other), equality.Rules{}) {
		t.Fatal("Equal reports two lists equal whose last links differ")
	}
}

// TestEqualAllocs checks the allocation ceiling of Equal on values without
// a map, and on a list of links, whose walk records the links it entered.
func TestEqualAllocs(t *testing.T) {
	values := []struct {
		name   string
		x, y   any
		allocs float64
	}{
		{name: "two ints", x: 1, y: 1, allocs: scalarAllocs},
		{name: "two slices of ints", x: []int{1, 2, 3}, y: []int{1, 2, 3}, allocs: sliceAllocs},
		{name: "two structs", x: hidden{1, 2}, y: hidden{1, 2}, allocs: structAllocs},
		{name: "two lists of three links", x: list(3, 2), y: list(3, 2), allocs: listAllocs},
	}
	for _, v := range values {
		x, y := reflect.ValueOf(v.x), reflect.ValueOf(v.y)
		got := testing.AllocsPerRun(allocRuns, func() { _ = equality.Equal(x, y, equality.Rules{}) })
		if matcher.AllocationsCounted() && got > v.allocs {
			t.Errorf("Equal of %s allocates %v times, want at most %v", v.name, got, v.allocs)
		}
	}
}

// BenchmarkEqual measures Equal on two slices of ints and on two lists of
// links.
func BenchmarkEqual(b *testing.B) {
	b.Run("slices", func(b *testing.B) {
		x, y := reflect.ValueOf([]int{1, 2, 3}), reflect.ValueOf([]int{1, 2, 3})
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Equal(x, y, equality.Rules{})
		}
	})

	b.Run("lists", func(b *testing.B) {
		x, y := reflect.ValueOf(list(3, 2)), reflect.ValueOf(list(3, 2))
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Equal(x, y, equality.Rules{})
		}
	})
}
