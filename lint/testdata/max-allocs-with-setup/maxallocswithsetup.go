// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package maxallocswithsetup

import (
	"testing"

	"go.dokimi.dev/assert"
)

func buffer() []byte { return make([]byte, 64) }

func decode(b []byte) {}

func correct(t *testing.T) {
	assert.MaxAllocsWithSetup(t, buffer, decode, 0, "Decode allocates nothing beside its buffer")
}

func counted(t *testing.T) {
	allocs := testing.AllocsPerRun(100, func() {
		decode(buffer())
	})
	if allocs > 1 { // want `max-allocs: state the check with MaxAllocs or MaxAllocsWithSetup`
		t.Fatalf("Decode allocates %v times", allocs)
	}
}
