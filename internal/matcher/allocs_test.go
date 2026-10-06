// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"os"
	"runtime/debug"
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

// TestMaxAllocs does not run in parallel: testing.AllocsPerRun panics
// while a parallel test runs.
func TestMaxAllocs(t *testing.T) {
	matchertest.RunMaxAllocs(t, func(s *matchertest.Seat, fn func(), ceiling uint64, msg string) {
		matcher.MaxAllocs(s, matcher.Fatal, fn, ceiling, msg)
	})
}

// TestMaxAllocsWithSetup does not run in parallel: its count covers the
// whole process.
func TestMaxAllocsWithSetup(t *testing.T) {
	matchertest.RunMaxAllocsWithSetup(t,
		func(s *matchertest.Seat, setup func() *[]byte, fn func(*[]byte), ceiling uint64, msg string) {
			matcher.MaxAllocsWithSetup(s, matcher.Fatal, setup, fn, ceiling, msg)
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

// countedHere returns what AllocationsCounted reports in the running
// binary, from its build information and its environment, and the build's
// settings. The build tags and the build information are two records of
// one build. Reading the second checks the first, in every build this
// suite runs under.
func countedHere() (bool, []debug.BuildSetting) {
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

// TestAllocationsCounted checks AllocationsCounted against the build
// information and the environment of the running binary, and in child
// processes whose environment states a variable of a mutation run. Each
// child reads its own environment on its first call.
func TestAllocationsCounted(t *testing.T) {
	t.Parallel()

	t.Run("agrees with the running binary's build information and environment", func(t *testing.T) {
		t.Parallel()

		want, settings := countedHere()
		if got := matcher.AllocationsCounted(); got != want {
			t.Fatalf("AllocationsCounted = %v, want %v for settings %v", got, want, settings)
		}
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
// its allocation ceiling, measured.
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
