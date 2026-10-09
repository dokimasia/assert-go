// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"context"
	"math/rand/v2"
	"runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// How long the independent check waits for goroutines on their way out,
// and how often it reads them meanwhile.
const (
	grace = 500 * time.Millisecond
	poll  = 10 * time.Millisecond
)

// referenceLabel is the key of the label that the independent check sets.
const referenceLabel = "matchertest.scope"

func TestGoroutine(t *testing.T) {
	t.Parallel()

	t.Run("RunNoGoroutineLeaks", func(t *testing.T) {
		t.Parallel()
		matchertest.RunNoGoroutineLeaks(t, leakCheck(slices.Sort))
	})
}

// TestGoroutineTwins runs TestGoroutineTwinsProcess in a child process, and
// requires the failure of RunNoGoroutineLeaks for a twin that reports the
// goroutines in descending order.
func TestGoroutineTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestGoroutineTwinsProcess",
		"want the functions of the 3 and 5 goroutines in ascending order")
}

// TestGoroutineTwinsProcess runs only in the child process of
// TestGoroutineTwins.
func TestGoroutineTwinsProcess(t *testing.T) {
	inChild(t)

	t.Run("RunNoGoroutineLeaks of a twin that reports in descending order", func(t *testing.T) {
		matchertest.RunNoGoroutineLeaks(t, leakCheck(func(leaked []string) {
			slices.Sort(leaked)
			slices.Reverse(leaked)
		}))
	})
}

// leakCheck returns a leak assertion that labels the calling goroutine with
// a scope of its own, and reports the function of each goroutine with that
// label that still runs after the grace, in the order that order puts them
// in.
func leakCheck(order func(leaked []string)) matchertest.LeakInvoke {
	return func(s *matchertest.Seat, msg string) func() {
		scope := strconv.FormatUint(rand.Uint64(), 36)
		pprof.SetGoroutineLabels(pprof.WithLabels(context.Background(), pprof.Labels(referenceLabel, scope)))

		return func() {
			pprof.SetGoroutineLabels(context.Background())
			var leaked []string
			for deadline := time.Now().Add(grace); ; time.Sleep(poll) {
				leaked = scoped(scope)
				if len(leaked) == 0 || time.Now().After(deadline) {
					break
				}
			}
			if len(leaked) > 0 {
				order(leaked)
				s.Report(matcher.Failure{
					Assertion: "no-task-leaks", Contract: msg,
					Detail: map[string]any{"leaked": leaked},
				}, true)
			}
		}
	}
}

// scoped returns the function of each goroutine whose header in a dump of
// every stack states the label of scope. The header states a goroutine's
// labels, because this module states go 1.27.2, whose traceback prints
// them. It reads the dump of the stacks and not the goroutine profile that
// the matcher reads, so the suite checks the matcher against a second
// implementation.
func scoped(scope string) []string {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]

	var out []string
	for block := range strings.SplitSeq(string(buf), "\n\n") {
		header, frames, _ := strings.Cut(block, "\n")
		if !strings.Contains(header, referenceLabel+": "+scope) {
			continue
		}
		function := ""
		for line := range strings.SplitSeq(frames, "\n") {
			if strings.HasPrefix(line, "created by ") {
				break
			}
			if i := strings.LastIndex(line, "("); i > 0 && !strings.HasPrefix(line, "\t") {
				function = line[:i]
			}
		}
		out = append(out, function)
	}
	return out
}
