// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// Not parallel, and neither are its cases: the reading is over the
// whole process. See [matchertest.RunNoGoroutineLeaks].
func TestGoroutine(t *testing.T) {
	t.Run("RunNoGoroutineLeaks", func(t *testing.T) {
		matchertest.RunNoGoroutineLeaks(t, func(s *matchertest.Seat, msg string) func() {
			before := ids()

			return func() {
				var leaked []uint64
				for id := range ids() {
					if !before[id] {
						leaked = append(leaked, id)
					}
				}
				if len(leaked) > 0 {
					s.Report(matcher.Failure{
						Assertion: "no-task-leaks", Contract: msg,
						Detail: map[string]any{"leaked": leaked},
					}, true)
				}
			}
		})
	})
}

// ids returns every live goroutine's id, which is what a leak check
// reads. Written out here rather than reused so the suite is driven by
// an independent implementation.
func ids() map[uint64]bool {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]

	out := map[uint64]bool{}
	for line := range strings.SplitSeq(string(buf), "\n") {
		rest, ok := strings.CutPrefix(line, "goroutine ")
		if !ok {
			continue
		}
		digits, _, ok := strings.Cut(rest, " ")
		if !ok {
			continue
		}
		if id, err := strconv.ParseUint(digits, 10, 64); err == nil {
			out[id] = true
		}
	}
	return out
}
