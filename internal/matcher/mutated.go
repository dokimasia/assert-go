// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"os"
	"sync"
)

// The variables that a mutation run sets in the environment of the test
// binaries that it runs.
const (
	// mutantVariable is set in every run of a mutation run: to 0 in a
	// control run, and to the active mutant's number in a mutant's run and
	// in the run that confirms a survivor.
	mutantVariable = "DOKIMI_MUTATE_MUTANT"
	// instrumentedVariable is set in every run of the test binary that
	// contains every mutant of a package behind a switch, the control runs
	// included, and in no run of an ordinary build.
	instrumentedVariable = "DOKIMI_MUTATE_INSTRUMENTED"
)

// Mutated reports whether a mutation run runs the test binary: whether
// DOKIMI_MUTATE_MUTANT is in its environment, whatever its value, the empty
// value included. A mutation run sets the variable in every run, its
// control runs and the ordinary build that confirms a survivor included, so
// state that the tests keep between runs is left unchanged.
//
// It reads the environment on its first call, and returns the result of
// that reading afterwards.
//
// # Allocation contract
//
// Mutated allocates nothing after its first call.
func Mutated() bool {
	return mutated()
}

// mutated computes [Mutated] on its first call.
var mutated = sync.OnceValue(func() bool {
	_, ok := os.LookupEnv(mutantVariable)
	return ok
})

// MutationInstrumented reports whether the running test binary is the one
// that a mutation run instrumented: whether DOKIMI_MUTATE_INSTRUMENTED is in
// its environment, whatever its value, the empty value included. That
// binary contains every mutant of a package behind a switch, so the
// compiler inlines fewer of its functions than in an ordinary build. The
// ordinary build of one mutant, which a mutation run builds to confirm a
// survivor, runs without the variable.
//
// It reads the environment on its first call, and returns the result of
// that reading afterwards.
//
// # Allocation contract
//
// MutationInstrumented allocates nothing after its first call.
func MutationInstrumented() bool {
	return mutationInstrumented()
}

// mutationInstrumented computes [MutationInstrumented] on its first call.
var mutationInstrumented = sync.OnceValue(func() bool {
	_, ok := os.LookupEnv(instrumentedVariable)
	return ok
})
