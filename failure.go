// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import "go.dokimi.dev/assert/internal/matcher"

// Failure is what a failing assertion reports.
//
// The record is the same shape in every implementation of the
// standard. The sentence a person reads is rendered from it and is not
// standardised, because each language reads its own conventions.
type Failure = matcher.Failure

// Where is the call site a failure came from.
//
// Line is zero when the frame could not be read, and a reader treats
// that as absent rather than as line zero.
type Where = matcher.Where

// Reporter is a [TB] that takes the record rather than the sentence.
//
// [testing.TB] declares three methods and can never grow a fourth, so
// the record reaches a seat through a second interface a seat may also
// satisfy. [Recorder] satisfies it and keeps every record;
// [testing.T] does not and receives the rendered sentence through
// Fatalf or Errorf.
//
// aborting is true for the aborting surface and false for the
// recording one, which is the same distinction Fatalf and Errorf carry.
type Reporter = matcher.Reporter
