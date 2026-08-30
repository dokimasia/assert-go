// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package matchertest holds every assertion's cases once, so each
// surface is proved against the same ones.
//
// A case says what an assertion is given and what it must report. The
// comparison lives in the matcher core, and three surfaces call it:
// the core itself, the aborting one, and the recording one. Testing
// each separately means writing the cases three times, and three
// copies drift.
//
// A surface supplies an invoker saying how it is called, and the suite
// drives every case through it:
//
//	func TestEqual(t *testing.T) {
//	    t.Parallel()
//	    matchertest.RunPair(t, matchertest.EqualCases(),
//	        func(s *matchertest.Seat, got, want any, msg string) {
//	            assert.Equal(s, got, want, msg)
//	        })
//	}
//
// A surface that stops reporting a failure, or reports a different
// one, fails the shared case rather than passing its own.
//
// # What a suite does not cover
//
// Cases state inputs and the failure they must produce. Anything about
// one surface alone belongs in that surface's own test: that a chain
// method returns its receiver, that an aborting seat stops, that a
// generated wrapper exists at all.
//
// # Dependency position
//
// Imports fmt, strings, sync and testing. Depends on no other package
// in this module, so every surface can import it.
package matchertest
