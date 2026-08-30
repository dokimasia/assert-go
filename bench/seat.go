// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package bench

import "go.dokimi.dev/assert"

// B is what a [Contract] requires of the benchmark it measures.
//
// It embeds [go.dokimi.dev/assert.TB] rather than replacing it: a
// benchmark reports a failure exactly as a test does, and only the
// iteration and metric reporting are its own. [testing.B] satisfies
// this as written.
//
// The recorder the assertion packages ship deliberately does not.
// Satisfying this would put benchmark machinery into the seat every
// ordinary test uses, to serve the one package that needs it.
type B interface {
	assert.TB

	// Loop reports whether the benchmark should run another
	// iteration, and is what [Contract.Loop] delegates to.
	Loop() bool

	// ReportMetric records a value under a unit, which is how a
	// contract publishes what it measured beside the ceiling.
	ReportMetric(n float64, unit string)
}
