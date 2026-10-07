// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package maxallocs

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func get() {}

func correct(t *testing.T) {
	assert.MaxAllocs(t, get, 0, "Get allocates nothing per call")
}

func counted(t *testing.T) {
	allocs := testing.AllocsPerRun(100, get)
	assert.Equal(t, allocs, 0.0, "Get allocates nothing per call")                    // want `max-allocs: state the check with MaxAllocs or MaxAllocsWithSetup of testing\.AllocsPerRun\(100, get\)`
	expect.True(t, testing.AllocsPerRun(10, get) <= 2, "Get allocates at most twice") // want `max-allocs: state the check with MaxAllocs or MaxAllocsWithSetup`
	if testing.AllocsPerRun(100, get) != 0 {                                          // want `max-allocs: state the check with MaxAllocs or MaxAllocsWithSetup`
		t.Fatal("Get allocates")
	}
}

func baseline(t *testing.T) {
	ceiling := uint64(testing.AllocsPerRun(100, get))
	assert.MaxAllocs(t, get, ceiling, "Get allocates at most what a baseline call allocates")
}
