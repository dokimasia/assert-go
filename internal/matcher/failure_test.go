// Copyright Dokimasia B.V. 2026
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
	field      any
	caseRecord matcher.Failure
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

	// The accessors of want and got read one field each, and share the
	// cases of a field that the record declares, declares as nil, or lacks.
	accessors := []struct {
		name  string
		field string
		read  func(matcher.Failure) (any, bool)
	}{
		{name: "Want", field: "want", read: matcher.Failure.Want},
		{name: "Got", field: "got", read: matcher.Failure.Got},
	}
	for _, a := range accessors {
		t.Run(a.name, func(t *testing.T) {
			t.Parallel()

			tests := []struct {
				name         string
				give         matcher.Failure
				wantValue    any
				wantDeclared bool
			}{
				{
					name:      "returns the field and that the record declares it",
					give:      matcher.Failure{Assertion: "equal", Detail: map[string]any{a.field: 7, "index": 3}},
					wantValue: 7, wantDeclared: true,
				},
				{
					name:      "returns a nil field that the record declares",
					give:      matcher.Failure{Assertion: "equal", Detail: map[string]any{a.field: nil}},
					wantValue: nil, wantDeclared: true,
				},
				{
					name:      "returns false for a record whose assertion declares another field",
					give:      matcher.Failure{Assertion: "has-prefix", Detail: map[string]any{"prefix": "x"}},
					wantValue: nil, wantDeclared: false,
				},
				{
					name:      "returns false for a record without a detail",
					give:      matcher.Failure{Assertion: "true", Contract: "the flag is set"},
					wantValue: nil, wantDeclared: false,
				},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()

					value, declared := a.read(tt.give)
					if value != tt.wantValue || declared != tt.wantDeclared {
						t.Fatalf("%s() = %v, %t, want %v, %t", a.name, value, declared, tt.wantValue, tt.wantDeclared)
					}
				})
			}
		})
	}

	t.Run("CaseFailure", func(t *testing.T) {
		t.Parallel()

		failing := matcher.Failure{
			Assertion: "equal",
			Contract:  "decoding returns the encoded values",
			Detail:    map[string]any{"want": 2, "got": 1},
		}
		tests := []struct {
			name   string
			give   matcher.Failure
			want   matcher.Failure
			wantOK bool
		}{
			{
				name:   "returns the record of the failing case that a property's record contains",
				give:   matcher.Failure{Assertion: "prop-for-all", Detail: map[string]any{"failure": failing}},
				want:   failing,
				wantOK: true,
			},
			{
				name: "returns false for a property's record whose failure is nil",
				give: matcher.Failure{Assertion: "prop-for-all", Detail: map[string]any{"failure": nil}},
			},
			{
				name: "returns false for a record whose assertion declares no failure",
				give: matcher.Failure{Assertion: "equal", Detail: map[string]any{"want": 2, "got": 1}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				record, ok := tt.give.CaseFailure()
				if record.Assertion != tt.want.Assertion || record.Contract != tt.want.Contract ||
					len(record.Detail) != len(tt.want.Detail) || ok != tt.wantOK {

					t.Fatalf("CaseFailure() = %+v, %t, want %+v, %t", record, ok, tt.want, tt.wantOK)
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
// with its allocation ceiling: CallerWhere of the frames of this call, the
// JSON of a record of two fields with a call site, its want and its got,
// and the failing case of a property's record that contains it.
func failureCases() []allocCase {
	var pcs [8]uintptr
	frames := pcs[:runtime.Callers(1, pcs[:])]
	f := matcher.Failure{
		Assertion: "equal", Contract: "the count is right", Detail: map[string]any{"want": 2, "got": 1},
		Where: matcher.Where{File: "/src/shop/cart_test.go", Line: 42},
	}
	property := matcher.Failure{Assertion: "prop-for-all", Detail: map[string]any{"failure": f}}
	return []allocCase{
		{name: "CallerWhere", call: func(matcher.Seat) { where = matcher.CallerWhere(frames) }, allocs: 2},
		{name: "Failure.MarshalJSON", call: func(matcher.Seat) { marshalled, _ = f.MarshalJSON() }, allocs: 33},
		{name: "Failure.Want", call: func(matcher.Seat) { field, _ = f.Want() }},
		{name: "Failure.Got", call: func(matcher.Seat) { field, _ = f.Got() }},
		{name: "Failure.CaseFailure", call: func(matcher.Seat) { caseRecord, _ = property.CaseFailure() }},
	}
}
