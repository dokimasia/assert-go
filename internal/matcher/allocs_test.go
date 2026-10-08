// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"flag"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The results of the allocation cases, which keep each call.
var (
	counted bool
	off     bool
)

// kept keeps what allocate builds, so escape analysis cannot remove the
// allocation.
var kept []byte

// allocate makes one heap allocation per call.
func allocate() { kept = make([]byte, 64) }

// The notes of a ceiling that only the test log leaves unchecked, each on
// the line of the test's call, as the log of a test states them.
var (
	unchecked          = regexp.MustCompile(`allocs_test\.go:\d+: max-allocs: the ceiling is not checked`)
	uncheckedWithSetup = regexp.MustCompile(`allocs_test\.go:\d+: max-allocs-with-setup: the ceiling is not checked`)
)

// TestMaxAllocs does not run in parallel: its count covers the whole
// process, and a case sets GOMAXPROCS.
func TestMaxAllocs(t *testing.T) {
	matchertest.RunMaxAllocs(t, func(s *matchertest.Seat, fn func(), ceiling uint64, msg string) {
		matcher.MaxAllocs(s, matcher.Fatal, fn, ceiling, msg)
	})

	t.Run("writes the record of a passing call", func(t *testing.T) {
		checkPassRecord(t, "max-allocs", func(seat matcher.Seat) {
			matcher.MaxAllocs(seat, matcher.Fatal, func() {}, 0, allocContract)
		})
	})

	t.Run("sets GOMAXPROCS to 1 while it counts, and restores it", func(t *testing.T) {
		defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(2))
		procs := 0
		matcher.MaxAllocs(&matchertest.Seat{}, matcher.Fatal,
			func() { procs = max(procs, runtime.GOMAXPROCS(0)) }, 0, allocContract)
		if got := runtime.GOMAXPROCS(0); procs != 1 || got != 2 {
			t.Fatalf("the calls ran with GOMAXPROCS of at most %d and left it at %d, want 1 and 2", procs, got)
		}
	})

	t.Run("notes and passes a call over its ceiling in a run that writes the test log", func(t *testing.T) {
		if childtest.InChild(t) {
			matcher.MaxAllocs(t, matcher.Fatal, allocate, 0, allocContract)
			return
		}
		built, settings := builtHere()
		out := runLogged(t)
		if noted := unchecked.MatchString(out); noted != built {
			t.Fatalf("the child notes the ceiling at its call: %v, want %v for settings %v:\n%s",
				noted, built, settings, out)
		}
	})

	t.Run("notes no ceiling in a run without the test log", func(t *testing.T) {
		if childtest.InChild(t) {
			matcher.MaxAllocs(t, matcher.Fatal, func() {}, 0, allocContract)
			return
		}
		if out := runChild(t); strings.Contains(out, "the ceiling is not checked") {
			t.Fatalf("the child notes an unchecked ceiling:\n%s", out)
		}
	})
}

// TestMaxAllocsWithSetupAllocs does not run in parallel: its count covers the
// whole process, and a case sets GOMAXPROCS.
func TestMaxAllocsWithSetupAllocs(t *testing.T) {
	matchertest.RunMaxAllocsWithSetup(t,
		func(s *matchertest.Seat, setup func() *[]byte, fn func(*[]byte), ceiling uint64, msg string) {
			matcher.MaxAllocsWithSetup(s, matcher.Fatal, setup, fn, ceiling, msg)
		})

	t.Run("writes the record of a passing call", func(t *testing.T) {
		checkPassRecord(t, "max-allocs-with-setup", func(seat matcher.Seat) {
			matcher.MaxAllocsWithSetup(seat, matcher.Fatal, func() int { return 0 }, func(int) {}, 0, allocContract)
		})
	})

	t.Run("sets GOMAXPROCS to 1 while it counts, and restores it", func(t *testing.T) {
		defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(2))
		procs := 0
		matcher.MaxAllocsWithSetup(&matchertest.Seat{}, matcher.Fatal, func() int { return 0 },
			func(int) { procs = max(procs, runtime.GOMAXPROCS(0)) }, 0, allocContract)
		if got := runtime.GOMAXPROCS(0); procs != 1 || got != 2 {
			t.Fatalf("the calls ran with GOMAXPROCS of at most %d and left it at %d, want 1 and 2", procs, got)
		}
	})

	t.Run("notes and passes a call over its ceiling in a run that writes the test log", func(t *testing.T) {
		if childtest.InChild(t) {
			matcher.MaxAllocsWithSetup(t, matcher.Fatal, func() int { return 0 }, func(int) { allocate() }, 0,
				allocContract)
			return
		}
		built, settings := builtHere()
		out := runLogged(t)
		if noted := uncheckedWithSetup.MatchString(out); noted != built {
			t.Fatalf("the child notes the ceiling at its call: %v, want %v for settings %v:\n%s",
				noted, built, settings, out)
		}
	})
}

// buildWith returns build information that records gcflags as its
// -gcflags setting.
func buildWith(gcflags string) *debug.BuildInfo {
	return &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "CGO_ENABLED", Value: "1"},
		{Key: "-gcflags", Value: gcflags},
	}}
}

// TestOptimisationsOff checks which -gcflags turn off optimisation or
// inlining.
func TestOptimisationsOff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give *debug.BuildInfo
		want bool
	}{
		{"reports false without build information", nil, false},
		{"reports false for build information without -gcflags", &debug.BuildInfo{}, false},
		{"reports false for flags that change neither", buildWith("-m -e"), false},
		{"reports false for a flag whose name starts with l", buildWith("-lang=go1.26"), false},
		{
			"reports false for -N and -l in a setting other than -gcflags",
			&debug.BuildInfo{Settings: []debug.BuildSetting{{Key: "-ldflags", Value: "-N -l"}}},
			false,
		},
		{"reports true for optimisation off", buildWith("-N"), true},
		{"reports true for inlining off", buildWith("-l"), true},
		{"reports true for both, for every package, as a debugger builds", buildWith("all=-N -l"), true},
		{"reports true for inlining off in one package", buildWith("example.com/pkg=-l"), true},
		{"reports true for inlining off with a level", buildWith("-l=4"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := matcher.OptimisationsOff(tt.give); got != tt.want {
				t.Fatalf("OptimisationsOff = %v, want %v", got, tt.want)
			}
		})
	}
}

// instrumentedSettings are the build settings that record the race
// detector, msan and asan.
var instrumentedSettings = map[string]bool{"-race": true, "-msan": true, "-asan": true}

// builtHere reports whether the build information and the environment of
// the running binary let its allocations count, and returns the build's
// settings. The build tags and the build information are two records of one
// build. Reading the second checks the first, in every build this suite runs
// under.
func builtHere() (bool, []debug.BuildSetting) {
	info, _ := debug.ReadBuildInfo()
	instrumented := false
	for _, setting := range info.Settings {
		if instrumentedSettings[setting.Key] && setting.Value == "true" {
			instrumented = true
		}
	}
	_, mutation := os.LookupEnv(instrumentedVariable)
	return !instrumented && !matcher.OptimisationsOff(info) && !mutation, info.Settings
}

// countedHere returns what AllocationsCounted reports in the running
// binary: what builtHere reports, in a run that writes no test log of go
// test. It returns the build's settings too.
func countedHere() (bool, []debug.BuildSetting) {
	built, settings := builtHere()
	return built && flag.Lookup("test.testlogfile").Value.String() == "", settings
}

// TestAllocationsCounted checks AllocationsCounted against the build
// information, the environment and the test log of the running binary, and
// in child processes whose environment states a variable of a mutation run
// or that write a test log. Each child reads its own environment on its
// first call.
func TestAllocationsCounted(t *testing.T) {
	t.Parallel()

	t.Run("agrees with the running binary's build information, environment and test log", func(t *testing.T) {
		t.Parallel()

		want, settings := countedHere()
		if got := matcher.AllocationsCounted(); got != want {
			t.Fatalf("AllocationsCounted = %v, want %v for settings %v", got, want, settings)
		}
	})

	t.Run("reports false in a run that writes the test log of go test", func(t *testing.T) {
		t.Parallel()

		if childtest.InChild(t) {
			if matcher.AllocationsCounted() {
				t.Fatal("AllocationsCounted = true, want false in a run that writes the test log of go test")
			}
			return
		}
		runLogged(t)
	})

	tests := []struct {
		name string
		give string
	}{
		{name: "reports false in a test binary that a mutation run instrumented", give: instrumentedVariable + "=1"},
		{
			name: "reports false whatever the variable states, the empty value included",
			give: instrumentedVariable + "=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if childtest.InChild(t) {
				if matcher.AllocationsCounted() {
					t.Fatal("AllocationsCounted = true, want false in a binary that a mutation run instrumented")
				}
				return
			}
			runChild(t, tt.give)
		})
	}

	t.Run("counts in the ordinary build that confirms a survivor of a mutation run", func(t *testing.T) {
		t.Parallel()

		if childtest.InChild(t) {
			want, settings := countedHere()
			if got := matcher.AllocationsCounted(); got != want {
				t.Fatalf("AllocationsCounted = %v, want %v for settings %v with %s set", got, want, settings,
					mutantVariable)
			}
			return
		}
		runChild(t, mutantVariable+"=12")
	})
}

// TestAllocsAllocs checks the allocation ceiling of a passing call of each
// function of allocs.go.
func TestAllocsAllocs(t *testing.T) {
	checkAllocs(t, allocsCases())
}

// BenchmarkAllocs measures a passing call of each function of allocs.go.
func BenchmarkAllocs(b *testing.B) {
	benchAllocs(b, allocsCases())
}

// allocsCases returns a passing call of each function of allocs.go, with
// its allocation ceiling.
func allocsCases() []allocCase {
	debugger := buildWith("all=-N -l")
	return []allocCase{
		{name: "MaxAllocs", call: func(seat matcher.Seat) {
			matcher.MaxAllocs(seat, matcher.Fatal, func() {}, 0, allocContract)
		}},
		{name: "MaxAllocsWithSetup", call: func(seat matcher.Seat) {
			matcher.MaxAllocsWithSetup(seat, matcher.Fatal, func() int { return 0 }, func(int) {}, 0, allocContract)
		}},
		{name: "AllocationsCounted", call: func(matcher.Seat) { counted = matcher.AllocationsCounted() }},
		{name: "OptimisationsOff", call: func(matcher.Seat) { off = matcher.OptimisationsOff(debugger) }},
	}
}
