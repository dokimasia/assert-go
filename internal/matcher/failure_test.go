// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"encoding/json"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
)

// The results of the allocation cases, which keep each call.
var (
	where      matcher.Where
	marshalled []byte
)

// TestFailure checks the call site of a failure and its JSON.
func TestFailure(t *testing.T) {
	t.Parallel()

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

	t.Run("MarshalJSON", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give matcher.Failure
			want string
		}{
			{
				name: "returns the record with typed literals and the base name of its file",
				give: matcher.Failure{
					Assertion: "equal", Contract: "the count is right", Detail: map[string]any{"want": 2, "got": 1},
					Where: matcher.Where{File: "/src/shop/cart_test.go", Line: 42},
				},
				want: `{"assertion":"equal","contract":"the count is right",` +
					`"detail":{"got":{"type":"int","value":1},"want":{"type":"int","value":2}},` +
					`"where":{"file":"cart_test.go","line":42}}`,
			},
			{
				name: "returns an empty detail and no site for a record without them",
				give: matcher.Failure{Assertion: "true", Contract: "the flag is set"},
				want: `{"assertion":"true","contract":"the flag is set","detail":{}}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				got, err := json.Marshal(tt.give)
				if err != nil || string(got) != tt.want {
					t.Fatalf("json.Marshal() = %s, %v, want %s", got, err, tt.want)
				}
			})
		}
	})
}

// TestFailureAllocs checks the allocation ceiling of each function and
// method of failure.go.
func TestFailureAllocs(t *testing.T) {
	checkAllocs(t, failureCases())
}

// BenchmarkFailure measures each function and method of failure.go.
func BenchmarkFailure(b *testing.B) {
	benchAllocs(b, failureCases())
}

// failureCases returns a call of each function and method of failure.go,
// with its allocation ceiling, measured: CallerWhere of the frames of this
// call, and the JSON of a record of two fields with a call site.
func failureCases() []allocCase {
	var pcs [8]uintptr
	frames := pcs[:runtime.Callers(1, pcs[:])]
	f := matcher.Failure{
		Assertion: "equal", Contract: "the count is right", Detail: map[string]any{"want": 2, "got": 1},
		Where: matcher.Where{File: "/src/shop/cart_test.go", Line: 42},
	}
	return []allocCase{
		{name: "CallerWhere", call: func(matcher.Seat) { where = matcher.CallerWhere(frames) }, allocs: 1},
		{name: "Failure.MarshalJSON", call: func(matcher.Seat) { marshalled, _ = f.MarshalJSON() }, allocs: 26},
	}
}
