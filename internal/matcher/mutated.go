// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"os"
	"sync"
)

// mutantVariable is the variable that a mutation run sets in the
// environment of every run of a test binary that it instrumented, its
// control runs included.
const mutantVariable = "DOKIMI_MUTATE_MUTANT"

// Mutated reports whether a mutation run instrumented the running test
// binary: whether DOKIMI_MUTATE_MUTANT is in its environment, whatever its
// value, the empty value included. A mutation run builds every mutant of a
// package into one test binary, behind a switch, and sets the variable in
// every run of the binary, its control runs included.
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
