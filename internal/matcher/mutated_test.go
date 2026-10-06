// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"os"
	"testing"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matcher"
)

// The results of the allocation cases, which keep each call.
var (
	mutatedBinary      bool
	instrumentedBinary bool
)

// TestMutated checks what a test binary reads of a mutation run from its
// environment: against the environment of the running binary, and in child
// processes whose environment states a variable of a mutation run. Each
// child reads its own environment on its first call.
func TestMutated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		variable string
		read     func() bool
		// ignores is a variable of a mutation run that the function does not
		// read, and empty for none.
		ignores string
	}{
		{name: "Mutated", variable: mutantVariable, read: matcher.Mutated},
		{
			name: "MutationInstrumented", variable: instrumentedVariable, read: matcher.MutationInstrumented,
			ignores: mutantVariable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("agrees with the running binary's environment", func(t *testing.T) {
				t.Parallel()

				_, want := os.LookupEnv(tt.variable)
				if got := tt.read(); got != want {
					t.Fatalf("%s = %v, want %v", tt.name, got, want)
				}
			})

			children := []struct {
				name string
				give string
			}{
				{name: "reports true for a value of its variable", give: tt.variable + "=12"},
				{name: "reports true for the empty value of its variable", give: tt.variable + "="},
			}
			for _, c := range children {
				t.Run(c.name, func(t *testing.T) {
					t.Parallel()

					if childtest.InChild(t) {
						if !tt.read() {
							t.Fatalf("%s = false, want true with %s", tt.name, c.give)
						}
						return
					}
					runChild(t, c.give)
				})
			}

			if tt.ignores == "" {
				return
			}
			t.Run("ignores the variable that a confirmation run sets", func(t *testing.T) {
				t.Parallel()

				if childtest.InChild(t) {
					_, want := os.LookupEnv(tt.variable)
					if got := tt.read(); got != want {
						t.Fatalf("%s = %v, want %v with %s set", tt.name, got, want, tt.ignores)
					}
					return
				}
				runChild(t, tt.ignores+"=12")
			})
		})
	}
}

// TestMutatedAllocs checks the allocation ceiling of a call of each
// function of mutated.go.
func TestMutatedAllocs(t *testing.T) {
	checkAllocs(t, mutatedCases())
}

// BenchmarkMutated measures a call of each function of mutated.go.
func BenchmarkMutated(b *testing.B) {
	benchAllocs(b, mutatedCases())
}

// mutatedCases returns a call of each function of mutated.go, with its
// allocation ceiling, measured.
func mutatedCases() []allocCase {
	return []allocCase{
		{name: "Mutated", call: func(matcher.Seat) { mutatedBinary = matcher.Mutated() }},
		{
			name: "MutationInstrumented",
			call: func(matcher.Seat) { instrumentedBinary = matcher.MutationInstrumented() },
		},
	}
}
