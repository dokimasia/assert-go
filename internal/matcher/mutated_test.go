// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"os"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matcher"
)

// mutatedBinary keeps the result of the allocation case of Mutated.
var mutatedBinary bool

// TestMutated checks Mutated against the environment of the running binary,
// and in child processes whose environment states the variable of a
// mutation run. Each child reads its own environment on its first call.
func TestMutated(t *testing.T) {
	t.Parallel()

	t.Run("agrees with the running binary's environment", func(t *testing.T) {
		t.Parallel()

		_, want := os.LookupEnv(mutantVariable)
		if got := matcher.Mutated(); got != want {
			t.Fatalf("Mutated = %v, want %v", got, want)
		}
	})

	tests := []struct {
		name string
		give string
	}{
		{name: "reports true in a test binary that runs a mutant", give: mutantVariable + "=12"},
		{name: "reports true whatever the variable states, the empty value included", give: mutantVariable + "="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if childtest.InChild(t) {
				if !matcher.Mutated() {
					t.Fatal("Mutated = false, want true in a binary that a mutation run instrumented")
				}
				return
			}
			out, err := childtest.Run(t, t.Name(), tt.give)
			if err != nil || !strings.Contains(out, "--- PASS: "+t.Name()+" ") {
				t.Fatalf("the child exits with %v, want a pass:\n%s", err, out)
			}
		})
	}
}

// TestMutatedAllocs checks the allocation ceiling of a call of Mutated.
func TestMutatedAllocs(t *testing.T) {
	checkAllocs(t, mutatedCases())
}

// BenchmarkMutated measures a call of Mutated.
func BenchmarkMutated(b *testing.B) {
	benchAllocs(b, mutatedCases())
}

// mutatedCases returns a call of Mutated, with its allocation ceiling,
// measured.
func mutatedCases() []allocCase {
	return []allocCase{
		{name: "Mutated", call: func(matcher.Seat) { mutatedBinary = matcher.Mutated() }},
	}
}
