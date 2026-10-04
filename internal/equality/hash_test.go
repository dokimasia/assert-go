// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/equality"
	"go.dokimi.dev/assert/internal/matcher"
)

// The allocations of Hash, measured.
const (
	// hashScalarAllocs are the allocations of Hash of an int.
	hashScalarAllocs = 0
	// hashSliceAllocs are the allocations of Hash of a slice of ints.
	hashSliceAllocs = 0
	// hashStructAllocs are the allocations of Hash of a struct of ints.
	hashStructAllocs = 0
	// hashMapAllocs are the allocations of Hash of a map of three ints: the
	// map's iterator, and the copies of keys and values that it returns.
	hashMapAllocs = 9
)

// mapRuns is the number of times a test hashes one map, each time over the
// entries in the order that one iteration of the map gives.
const mapRuns = 64

// long is the length of a slice and of a string past the bounds of the walk
// of Hash, which reads 256 parts and 256 bytes.
const long = 300

// tailed is a struct whose map is followed by a field that the walk of Hash
// reads only when the map leaves it parts.
type tailed struct {
	M    map[int]int
	Tail int
}

// counted returns a map of n entries with the keys from first, each mapped
// to value.
func counted(n, first, value int) map[int]int {
	m := make(map[int]int, n)
	for k := range n {
		m[first+k] = value
	}
	return m
}

// counting returns a slice of long ints from 0, with the element at index i
// set to value.
func counting(i, value int) []int {
	s := make([]int, long)
	for k := range s {
		s[k] = k
	}
	s[i] = value
	return s
}

// textOf returns a string of long bytes 'a', with the byte at index i set to
// b.
func textOf(i int, b byte) string {
	s := []byte(strings.Repeat("a", long))
	s[i] = b
	return string(s)
}

// deep returns v inside levels slices, each the one element of the next,
// so that v is at the depth levels of the walk.
func deep(levels int, v any) any {
	out := v
	for range levels {
		out = []any{out}
	}
	return out
}

// pointers returns v behind levels pointers, each to the next, so that v is
// at the depth levels of the walk.
func pointers(levels int, v any) any {
	out := v
	for range levels {
		target := out
		out = &target
	}
	return out
}

// hash returns the hash of x.
func hash(x any) uint64 {
	return equality.Hash(reflect.ValueOf(x))
}

// TestHash checks that Hash agrees with Equal, reads a value within its
// bounds, and tells apart the values that differ within them.
func TestHash(t *testing.T) {
	t.Parallel()

	negativeZero := math.Copysign(0, -1)
	selfLoop := &link{ID: 1}
	selfLoop.Next = selfLoop
	otherSelfLoop := &link{ID: 1}
	otherSelfLoop.Next = otherSelfLoop
	twoLoop := &link{ID: 1, Next: &link{ID: 1}}
	twoLoop.Next.Next = twoLoop
	ascending, descending := map[int]string{}, map[int]string{}
	for k := range 100 {
		ascending[k] = "v"
		descending[99-k] = "v"
	}
	channel := make(chan int)

	t.Run("Hash", func(t *testing.T) {
		t.Parallel()

		alike := []struct {
			name  string
			giveX any
			giveY any
		}{
			{name: "hashes -0 as +0", giveX: negativeZero, giveY: 0.0},
			{name: "hashes a float32 -0 as +0", giveX: float32(negativeZero), giveY: float32(0)},
			{
				name:  "hashes the zeros of a complex number alike",
				giveX: complex(negativeZero, negativeZero),
				giveY: complex(0, 0),
			},
			{name: "hashes pointers to equal values alike", giveX: new(1), giveY: new(1)},
			{name: "hashes two lists that contain themselves alike", giveX: selfLoop, giveY: otherSelfLoop},
			{name: "hashes a cycle of one link as a cycle of two links alike", giveX: selfLoop, giveY: twoLoop},
			{name: "hashes two closures of one literal alike", giveX: counter(1), giveY: counter(2)},
			{name: "hashes structs of equal unexported fields alike", giveX: hidden{1, 2}, giveY: hidden{1, 2}},
			{
				name:  "hashes equal values inside interfaces alike",
				giveX: holder{V: []any{1, "a"}},
				giveY: holder{V: []any{1, "a"}},
			},
			{name: "hashes two maps of equal entries alike", giveX: ascending, giveY: descending},
			{
				name:  "reads no part past the 256th",
				giveX: counting(255, -1),
				giveY: counting(255, -2),
			},
			{name: "reads no level past the 8th", giveX: deep(8, 1), giveY: deep(8, 2)},
			{name: "reads no level past the 8th behind pointers", giveX: pointers(8, 1), giveY: pointers(8, 2)},
			{
				name:  "reads no key of a map at the 7th level",
				giveX: deep(7, map[int]int{1: 1}),
				giveY: deep(7, map[int]int{2: 1}),
			},
			{
				name:  "reads no value of a map at the 7th level",
				giveX: deep(7, map[int]int{1: 1}),
				giveY: deep(7, map[int]int{1: 2}),
			},
			{
				name:  "reads no byte of a string past the 256th",
				giveX: textOf(256, 'b'),
				giveY: textOf(256, 'c'),
			},
			{
				name:  "hashes a map of more entries than parts left by its length",
				giveX: counted(300, 0, 1),
				giveY: counted(300, 1000, 1),
			},
			{
				name:  "reads the keys alone of a map whose entries take the parts left one each",
				giveX: counted(255, 0, 1),
				giveY: counted(255, 0, 2),
			},
			{
				name:  "reads no part after a map that took the parts left",
				giveX: tailed{M: counted(2, 0, 1), Tail: 1},
				giveY: tailed{M: counted(2, 0, 1), Tail: 2},
			},
		}
		for _, tt := range alike {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				if x, y := hash(tt.giveX), hash(tt.giveY); x != y {
					t.Fatalf("Hash returns %#x and %#x, want one hash", x, y)
				}
			})
		}

		different := []struct {
			name  string
			giveX any
			giveY any
		}{
			{name: "hashes two ints apart", giveX: 1, giveY: 2},
			{name: "hashes two unsigned integers apart", giveX: uint8(7), giveY: uint8(8)},
			{name: "hashes true and false apart", giveX: true, giveY: false},
			{name: "hashes two floats apart", giveX: 1.5, giveY: 2.5},
			{name: "hashes complex numbers of other real parts apart", giveX: complex(1, 2), giveY: complex(3, 2)},
			{name: "hashes complex numbers of other imaginary parts apart", giveX: complex(1, 2), giveY: complex(1, 3)},
			{name: "hashes two strings apart", giveX: "a", giveY: "b"},
			{name: "hashes strings of other lengths apart", giveX: "a", giveY: "aa"},
			{
				name:  "hashes strings that differ in the 256th byte apart",
				giveX: textOf(255, 'b'),
				giveY: textOf(255, 'c'),
			},
			{name: "hashes two channels apart", giveX: channel, giveY: make(chan int)},
			{name: "hashes two functions apart", giveX: one, giveY: two},
			{name: "hashes a nil pointer and a pointer apart", giveX: (*int)(nil), giveY: new(0)},
			{name: "hashes pointers to other values apart", giveX: new(1), giveY: new(2)},
			{name: "hashes a nil interface and a nil pointer apart", giveX: nil, giveY: (*int)(nil)},
			{name: "hashes arrays of another element apart", giveX: [2]int{1, 2}, giveY: [2]int{1, 3}},
			{name: "hashes structs of another unexported field apart", giveX: hidden{1, 2}, giveY: hidden{1, 3}},
			{
				name:  "hashes values of another value inside an interface apart",
				giveX: holder{V: 1},
				giveY: holder{V: 2},
			},
			{name: "hashes a nil slice and an empty one apart", giveX: []int(nil), giveY: []int{}},
			{name: "hashes slices of another element apart", giveX: []int{1, 2}, giveY: []int{1, 3}},
			{name: "reads the 256th part", giveX: counting(254, -1), giveY: counting(254, -2)},
			{name: "reads the 8th level", giveX: deep(7, 1), giveY: deep(7, 2)},
			{name: "reads the 8th level behind pointers", giveX: pointers(7, 1), giveY: pointers(7, 2)},
			{
				name:  "reads the keys of a map at the 6th level",
				giveX: deep(6, map[int]int{1: 1}),
				giveY: deep(6, map[int]int{2: 1}),
			},
			{
				name:  "reads the values of a map at the 6th level",
				giveX: deep(6, map[int]int{1: 1}),
				giveY: deep(6, map[int]int{1: 2}),
			},
			{name: "hashes a nil map and an empty one apart", giveX: map[int]int(nil), giveY: map[int]int{}},
			{name: "hashes maps of other keys apart", giveX: map[int]int{1: 1}, giveY: map[int]int{2: 1}},
			{name: "hashes maps of other values apart", giveX: map[int]int{1: 1}, giveY: map[int]int{1: 2}},
			{name: "hashes maps of other lengths apart", giveX: counted(300, 0, 1), giveY: counted(301, 0, 1)},
			{
				name:  "reads each key of a map whose entries take the parts left one each",
				giveX: counted(255, 0, 1),
				giveY: counted(255, 1000, 1),
			},
		}
		for _, tt := range different {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				if x, y := hash(tt.giveX), hash(tt.giveY); x == y {
					t.Fatalf("Hash returns %#x for both, want two hashes", x)
				}
			})
		}

		t.Run("hashes a map alike over every order of its entries", func(t *testing.T) {
			t.Parallel()
			m := counted(100, 0, 1)
			first := hash(m)
			for range mapRuns {
				if got := hash(m); got != first {
					t.Fatalf("Hash returns %#x, and %#x before, want one hash for one map", got, first)
				}
			}
		})
		t.Run("hashes a nil interface as the invalid value", func(t *testing.T) {
			t.Parallel()
			field := reflect.ValueOf(holder{}).Field(0)
			if x, y := equality.Hash(field), equality.Hash(reflect.Value{}); x != y {
				t.Fatalf("Hash returns %#x and %#x, want one hash", x, y)
			}
		})
	})
}

// TestHashAllocs checks the allocation ceiling of Hash on values without a
// map, and on a map, whose entries it reads through an iterator.
func TestHashAllocs(t *testing.T) {
	values := []struct {
		name   string
		x      any
		allocs float64
	}{
		{name: "an int", x: 1, allocs: hashScalarAllocs},
		{name: "a slice of ints", x: []int{1, 2, 3}, allocs: hashSliceAllocs},
		{name: "a struct", x: hidden{1, 2}, allocs: hashStructAllocs},
		{name: "a map of three ints", x: counted(3, 0, 1), allocs: hashMapAllocs},
	}
	for _, v := range values {
		x := reflect.ValueOf(v.x)
		got := testing.AllocsPerRun(allocRuns, func() { _ = equality.Hash(x) })
		if matcher.AllocationsCounted() && got > v.allocs {
			t.Errorf("Hash of %s allocates %v times, want at most %v", v.name, got, v.allocs)
		}
	}
}

// BenchmarkHash measures Hash on a struct and on a map.
func BenchmarkHash(b *testing.B) {
	b.Run("struct", func(b *testing.B) {
		x := reflect.ValueOf(hidden{1, 2})
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Hash(x)
		}
	})

	b.Run("map", func(b *testing.B) {
		x := reflect.ValueOf(counted(3, 0, 1))
		b.ReportAllocs()
		for b.Loop() {
			_ = equality.Hash(x)
		}
	})
}
