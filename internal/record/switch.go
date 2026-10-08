// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package record

import (
	"errors"
	"os"
	"sync"

	"go.dokimi.dev/assert/internal/fault"
)

// Variable is the environment variable that switches recording on.
const Variable = "DOKIMI_ASSERT_RECORD"

// ErrSwitch reports a value of [Variable] other than an empty one, 0 and 1.
var ErrSwitch = errors.New("record: the switch states neither 0 nor 1")

// switched is the reading of Variable, once per process.
var switched = sync.OnceValues(func() (bool, error) {
	switch value := os.Getenv(Variable); value {
	case "", "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fault.In("record.On", fault.At(
			fault.Of(ErrSwitch, "%q is neither 0 nor 1", value), fault.Field(Variable)))
	}
})

// On reports whether a test's seat writes call records: false for an
// unset, empty or 0 DOKIMI_ASSERT_RECORD, and true for 1. It reads the
// variable once per process, so a change during a run has no effect.
//
// # Errors
//
// It returns false and a fault of the kind [ErrSwitch] at the variable for
// any other value.
//
// # Allocation contract
//
// On allocates nothing after its first call.
func On() (bool, error) {
	return switched()
}
