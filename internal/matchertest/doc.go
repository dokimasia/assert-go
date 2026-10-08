// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package matchertest states every assertion's cases once, so each
// surface is proved against the same ones.
//
// A case states what an assertion is given and what it must report. The
// comparison is in the matcher core, and three surfaces call it: the core
// itself, the aborting one, and the recording one. A copy of the cases for
// each surface would need three edits for every change of a case.
//
// A surface supplies an invoker that states how it is called, and the
// suite drives every case through it:
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
// one, fails the shared case.
//
// # What a suite does not cover
//
// Cases state inputs and the failure they must produce. A behaviour of one
// surface alone is in that surface's own test: that a chain method returns
// its receiver, that an aborting seat stops, and that a generated wrapper
// exists.
//
// # Dependency position
//
// Imports internal/matcher for the failure record, the writer's text of a
// fault and the allocation flag, and the standard library. It compares a
// record's detail with a comparison of its own, and imports no surface, so
// every surface can import it.
package matchertest
