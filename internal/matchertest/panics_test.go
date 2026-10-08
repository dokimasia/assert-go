// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest_test

import (
	"testing"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

func TestPanics(t *testing.T) {
	t.Parallel()

	t.Run("RunPanics", func(t *testing.T) {
		t.Parallel()

		matchertest.RunPanics(t, func(s *matchertest.Seat, fn func(), msg string) (recovered any) {
			panicked := true
			func() {
				defer func() { recovered = recover() }()
				fn()
				panicked = false
			}()
			if !panicked {
				s.Report(matcher.Failure{Assertion: "throws", Contract: msg}, true)
			}
			return recovered
		})
	})

	t.Run("RunNotPanics", func(t *testing.T) {
		t.Parallel()

		matchertest.RunNotPanics(t, func(s *matchertest.Seat, fn func(), msg string) {
			defer func() {
				if r := recover(); r != nil {
					s.Report(matcher.Failure{
						Assertion: "not-throws", Contract: msg,
						Detail: map[string]any{"got": r},
					}, true)
				}
			}()
			fn()
		})
	})
}

// TestPanicsTwins runs TestPanicsTwinsProcess in a child process, and
// requires the failures of RunPanics and RunNotPanics for twins that
// recover the panic silently.
func TestPanicsTwins(t *testing.T) {
	t.Parallel()
	expectBroken(t, "TestPanicsTwinsProcess",
		`yielded <nil>, want "matchertest: the stated reason"`,
		"reported nothing for a panicking function")
}

// TestPanicsTwinsProcess runs only in the child process of TestPanicsTwins.
func TestPanicsTwinsProcess(t *testing.T) {
	inChild(t)

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
}
