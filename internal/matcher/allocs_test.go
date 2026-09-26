// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher_test

import (
	"runtime/debug"
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// TestMaxAllocs does not run in parallel: testing.AllocsPerRun panics
// while a parallel test runs.
func TestMaxAllocs(t *testing.T) {
	matchertest.RunMaxAllocs(t, func(s *matchertest.Seat, fn func(), ceiling uint64, msg string) {
		matcher.MaxAllocs(s, matcher.Fatal, fn, ceiling, msg)
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

func TestOptimisationsOff(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		info *debug.BuildInfo
		want bool
	}{
		{"no build information", nil, false},
		{"no -gcflags recorded", &debug.BuildInfo{}, false},
		{"flags that change nothing", buildWith("-m -e"), false},
		{"a flag whose name starts with l", buildWith("-lang=go1.26"), false},
		{"optimisation off", buildWith("-N"), true},
		{"inlining off", buildWith("-l"), true},
		{"both, for every package, as a debugger builds", buildWith("all=-N -l"), true},
		{"inlining off for one package", buildWith("example.com/pkg=-l"), true},
		{"inlining off with a level", buildWith("-l=4"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := matcher.OptimisationsOff(tc.info); got != tc.want {
				t.Fatalf("OptimisationsOff = %v, want %v", got, tc.want)
			}
		})
	}
}

// instrumentedSettings are the build settings that record the race
// detector, msan and asan.
var instrumentedSettings = map[string]bool{"-race": true, "-msan": true, "-asan": true}

func TestAllocationsCounted(t *testing.T) {
	t.Parallel()

	t.Run("agrees with the running binary's build information", func(t *testing.T) {
		t.Parallel()

		// The build tags and the build information are two records of one
		// build. Reading the second checks the first, in every build this
		// suite runs under.
		info, _ := debug.ReadBuildInfo()
		instrumented := false
		for _, setting := range info.Settings {
			if instrumentedSettings[setting.Key] && setting.Value == "true" {
				instrumented = true
			}
		}

		want := !instrumented && !matcher.OptimisationsOff(info)
		if got := matcher.AllocationsCounted(); got != want {
			t.Fatalf("AllocationsCounted = %v, want %v for settings %v", got, want, info.Settings)
		}
	})
}
