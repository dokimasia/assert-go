// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality_test

import (
	"reflect"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/equality"
)

// children is the number of processes that each case starts. The time
// package fills the fields of time.Local once in each process, so a process
// can race once.
const children = 10

// stamp is the second of the local times that the walks read.
const stamp = 1_700_000_000

// diffLimit is the most differences that the walk of Diff finds.
const diffLimit = 10

// TestLocal checks that each walk reads the fields of time.Local after the
// time package fills them. The time package fills them once in each process,
// so each case runs its walk in new processes of the test binary, beside the
// first formatting of a local time. Under the race detector, a process whose
// walk reads a field before the time package fills it fails.
func TestLocal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		walk    func() bool
		failure string
	}{
		{
			name: "Equal",
			walk: func() bool {
				return equality.Equal(localTime(0), localTime(0), equality.Rules{})
			},
			failure: "Equal reports two equal local times unequal",
		},
		{
			name: "Diff",
			walk: func() bool {
				return len(equality.Diff(localTime(0), localTime(1), equality.Rules{}, diffLimit)) == 1
			},
			failure: "Diff does not find the one difference of two local times",
		},
		{
			name: "Hash",
			walk: func() bool {
				x, y := localTime(0), localTime(0)
				return equality.Hash(x) == equality.Hash(y)
			},
			failure: "Hash returns other hashes for two equal local times",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			t.Run("reads time.Local after the time package fills it", func(t *testing.T) {
				t.Parallel()
				if childtest.InChild(t) {
					beside(t, tt.walk, tt.failure)
					return
				}
				for range children {
					if out, err := childtest.Run(t, t.Name()); err != nil {
						t.Fatalf("a process that runs %s beside the first formatting of a local time fails: %v\n%s",
							tt.name, err, out)
					}
				}
				if !tt.walk() {
					t.Fatal(tt.failure)
				}
			})
		})
	}
}

// localTime returns the local time offset seconds after stamp.
func localTime(offset int64) reflect.Value {
	return reflect.ValueOf(time.Unix(stamp+offset, 0))
}

// beside runs walk beside the first formatting of a local time in the
// process, and fails with failure when walk reports false.
func beside(t *testing.T, walk func() bool, failure string) {
	t.Helper()
	t.Run("formats a local time", func(t *testing.T) {
		t.Parallel()
		_ = time.Unix(stamp, 0).String()
	})
	t.Run("walks two local times", func(t *testing.T) {
		t.Parallel()
		if !walk() {
			t.Fatal(failure)
		}
	})
}
