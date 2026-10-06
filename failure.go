// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Failure is what a failing assertion reports.
//
// Every implementation of the definition reports the same record. The
// writer renders the sentence that a person reads from the record, and
// the definition leaves that sentence to each language.
//
// Want and Got return the fields want and got of the record's detail, and
// CaseFailure returns the record of a property's failing case. Each also
// reports whether the record contains what it returns:
//
//	if inner, ok := failure.CaseFailure(); ok {
//	    failure = inner
//	}
//	want, _ := failure.Want()
type Failure = matcher.Failure

// Where is the call site a failure came from.
//
// Line is zero when the frame could not be read, and a reader treats
// that as absent rather than as line zero.
type Where = matcher.Where

// Reporter is a [TB] that takes the record rather than the sentence.
//
// A seat receives the record through this second interface, so [TB]
// keeps the three methods that [testing.T] and [testing.B] implement.
// [Recorder] satisfies it and keeps every record. [testing.T] does not
// satisfy it, and receives the writer's text of the record through
// Fatalf or Errorf.
//
// aborting is true for the aborting surface and false for the recording
// one, as the choice between Fatalf and Errorf is.
type Reporter = matcher.Reporter
