// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The child that TestBrokenTwins runs to drive each suite with a twin
// that breaks it.
const (
	// brokenEnv makes TestBrokenTwinsChild run the broken twins.
	brokenEnv = "MATCHERTEST_BROKEN_TWINS"
	// brokenDeadline bounds the child process.
	brokenDeadline = time.Minute
)

// brokenFailures are the failures that the suites must report for the
// broken twins, one for each way a suite fails a twin.
var brokenFailures = []string{
	"the suite has no cases; it would pass having checked nothing",
	"the fixture allocated nothing, so the case checked no ceiling",
	"called the callable 0 times, want 101",
	"recovered <nil>, want the subject's own panic value",
	"the call did not run: hidden = 0, want 1",
	"returned matchertest: typed error, want the error from the chain",
	"returned nil for an error already of the target type",
	"returned matchertest: typed error on failure, want the zero value",
	`yielded <nil>, want "matchertest: the stated reason"`,
	"reported nothing for a panicking function",
	"matchertest: reported nothing, want a failure",
	"ran 0 attempts, want at least 3; it did not retry",
	"the body never ran",
	`the record contains the fields ["got" "index" "want"], want ["got" "want"]`,
	"the assertion returned after a callable ended its goroutine",
	"for a callable that ended its goroutine",
}

// TestBrokenTwinsChild runs only in the child process. Each twin breaks
// the suite that it drives, so the child fails.
func TestBrokenTwinsChild(t *testing.T) {
	if os.Getenv(brokenEnv) != "1" {
		t.Skip("runs only as the child of TestBrokenTwins")
	}

	t.Run("RunOne over no cases", func(t *testing.T) {
		matchertest.RunOne(t, nil, func(*matchertest.Seat, any, string) {})
	})

	t.Run("RunMaxAllocs of a twin that never calls the callable", func(t *testing.T) {
		matchertest.RunMaxAllocs(t, func(*matchertest.Seat, func(), uint64, string) {})
	})

	t.Run("RunCompletesWithin of a twin that recovers the subject's panic", func(t *testing.T) {
		matchertest.RunCompletesWithin(t, func(seat *matchertest.Seat, within time.Duration,
			fn func(ctx context.Context) error, msg string,
		) {
			matcher.CompletesWithin(seat, matcher.Fatal, within, func(ctx context.Context) error {
				defer func() { _ = recover() }()
				return fn(ctx)
			}, msg)
		})
	})

	t.Run("RunPure of a twin that never calls the subject", func(t *testing.T) {
		matchertest.RunPure(t, func(_ *matchertest.Seat, observe func() []int, _ func(), _ string) {
			_ = observe()
		})
	})

	t.Run("RunErrorAs of a twin that returns nil", func(t *testing.T) {
		matchertest.RunErrorAs(t, func(*matchertest.Seat, error, string) *matchertest.TypedError { return nil })
	})

	t.Run("RunErrorAs of a twin that returns a zero error", func(t *testing.T) {
		matchertest.RunErrorAs(t, func(*matchertest.Seat, error, string) *matchertest.TypedError {
			return &matchertest.TypedError{}
		})
	})

	t.Run("RunPanics of a twin that yields nothing", func(t *testing.T) {
		matchertest.RunPanics(t, func(_ *matchertest.Seat, fn func(), _ string) any {
			defer func() { _ = recover() }()
			fn()
			return nil
		})
	})

	t.Run("RunNotPanics of a twin that recovers silently", func(t *testing.T) {
		matchertest.RunNotPanics(t, func(_ *matchertest.Seat, fn func(), _ string) {
			defer func() { _ = recover() }()
			fn()
		})
	})

	t.Run("RunEventually of a twin that never runs the body", func(t *testing.T) {
		matchertest.RunEventually(t, func(*matchertest.Seat, time.Duration, time.Duration, func() bool, string) {})
	})

	t.Run("RunPermutation of a twin that reports an undeclared field", func(t *testing.T) {
		matchertest.RunPermutation(t, func(s *matchertest.Seat, got, want []any, msg string, opts ...matcher.Option) {
			if !permutes(got, want, opts) {
				report(s, "permutation", msg, map[string]any{"want": want, "got": got, "index": 0})
			}
		})
	})

	t.Run("RunPoisoned of a twin that induces on a goroutine of its own", func(t *testing.T) {
		matchertest.RunPoisoned(t, func(_ *matchertest.Seat, induce func(), _ func() error, _ string) {
			ended := make(chan struct{})
			go func() {
				defer close(ended)
				defer func() { _ = recover() }()
				induce()
			}()
			<-ended
		})
	})

	t.Run("RunPoisoned of a twin that reports as its goroutine ends", func(t *testing.T) {
		matchertest.RunPoisoned(t, func(s *matchertest.Seat, induce func(), _ func() error, msg string) {
			defer func() { _ = recover() }()
			defer report(s, "poisoned", msg, map[string]any{"index": nil, "got": nil})
			induce()
		})
	})
}

// TestBrokenTwins runs TestBrokenTwinsChild in a child process, and
// requires each failure of brokenFailures in its output.
//
// Under go test -cover the child writes its coverage counters to the
// directory of GOCOVERDIR, whose files the parent merges into its profile.
func TestBrokenTwins(t *testing.T) {
	t.Parallel()

	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("the test binary's path: %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), brokenDeadline)
	defer cancel()

	args := []string{"-test.run=^TestBrokenTwinsChild$", "-test.v", "-test.timeout=" + brokenDeadline.String()}
	if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
		args = append(args, "-test.gocoverdir="+coverDir)
	}
	child := exec.CommandContext(ctx, executable, args...)
	child.Env = append(os.Environ(), brokenEnv+"=1")
	out, err := child.CombinedOutput()

	if err == nil {
		t.Fatalf("the child passed, want each broken twin to fail:\n%s", out)
	}
	for _, failure := range brokenFailures {
		if !strings.Contains(string(out), failure) {
			t.Errorf("the child reported no %q:\n%s", failure, out)
		}
	}
}
