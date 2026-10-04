// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"fmt"
	"runtime"
	"sync"

	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/record"
)

// calls is the record of the calls that a Recorder receives.
type calls = record.Calls

// Recorder is a [TB] that records a failure instead of stopping the
// test. A test of an assertion reads from it what the assertion reported.
//
// A Recorder keeps the first fatal message, because nothing after that
// call runs in a test. It keeps every message that is not fatal, as a test
// that continues reports each one.
//
// [Recorder.Records] returns the call record of every assertion call that
// a Recorder receives.
//
// Call [NewRecorder] for a Recorder: the zero value is not usable.
//
// # Concurrency
//
// Every method is safe for concurrent use, so a driven body may run on a
// goroutine of its own.
type Recorder struct {
	calls

	mu sync.Mutex
	// goexit makes Fatalf end the calling goroutine once it has
	// recorded, so a driven body stops where a test would.
	goexit bool
	// failed records that some failure arrived, fatal or not.
	failed bool
	// fatal records that a fatal failure arrived, which fixes msg
	// against every later call.
	fatal bool
	// msg is the first fatal message, or the first non-fatal one when
	// nothing fatal has arrived.
	msg string
	// errors contains every non-fatal message in call order.
	errors []string
	// records contains every failure that arrived as a record, in call
	// order, whether fatal or not. An assertion reports one, and a
	// message passed to Fatalf or Errorf directly leaves none.
	records []Failure
	// helpers counts Helper calls, so a test can check an assertion
	// marks its own frame.
	helpers int
	// clock is what assertions read time from, or nil for the
	// platform clock.
	clock matcher.Clock
}

// NewRecorder returns a Recorder that records failures and returns
// from [Recorder.Fatalf]. Pair it with [Recorder.WithGoexit] where the
// driven body must stop instead.
//
// # Allocation contract
//
// NewRecorder allocates once: the Recorder.
func NewRecorder() *Recorder {
	r := &Recorder{}
	record.Keep(&r.calls)
	return r
}

// WithGoexit makes [Recorder.Fatalf] call [runtime.Goexit] once it has
// recorded, and returns the receiver so the call chains onto
// [NewRecorder].
//
// A body that runs on such a Recorder stops at the first assertion that
// it fails. Goexit ends the calling goroutine, so the body must run on a
// goroutine of its own.
//
// # Allocation contract
//
// WithGoexit allocates nothing.
func (r *Recorder) WithGoexit() *Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.goexit = true
	return r
}

// Report records one failure and, when it is fatal, stops the driven
// body as [Recorder.Fatalf] does.
//
// Report makes a Recorder a [Reporter]: an assertion passes it the
// record, so a test can read the assertion's own fields instead of
// searching its sentence for words. The Recorder keeps the writer's text
// of the record too, which [Recorder.Message] returns.
//
// # Allocation contract
//
// Report of a record that is not fatal and has no detail allocates twice
// on average. The Recorder keeps every record and every message, so its
// memory grows with each call.
func (r *Recorder) Report(f Failure, aborting bool) {
	r.mu.Lock()
	r.records = append(r.records, f)
	r.mu.Unlock()

	if aborting {
		r.Fatalf("%s", matcher.Render(f))
		return
	}
	r.Errorf("%s", matcher.Render(f))
}

// Failures returns every record that arrived, in call order. The slice
// is a fresh copy, so a caller may keep or sort it while the Recorder
// keeps recording.
//
// # Allocation contract
//
// Failures allocates once: the copy.
func (r *Recorder) Failures() []Failure {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]Failure(nil), r.records...)
}

// Records returns the call record of every assertion call that the
// Recorder received, one line of JSON each, in the order of their
// numbers. A call in a body that ran on the Recorder, such as an attempt
// of [Eventually] or a case of a property, is among them. The Recorder
// keeps the records whatever DOKIMI_ASSERT_RECORD states, and writes none
// of them to a test's output. The slice is a fresh copy.
//
// # Allocation contract
//
// Records allocates once: the copy.
func (r *Recorder) Records() []string {
	return record.Lines(&r.calls)
}

// Clock returns the clock that this Recorder gives assertions, which is
// the platform clock unless [Recorder.WithClock] set another.
//
// # Allocation contract
//
// Clock allocates nothing.
func (r *Recorder) Clock() matcher.Clock {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.clock == nil {
		return matcher.System{}
	}
	return r.clock
}

// WithClock makes assertions reported through this Recorder read c
// instead of the platform clock, and returns the receiver so the call
// chains onto [NewRecorder].
//
// # Allocation contract
//
// WithClock allocates nothing.
func (r *Recorder) WithClock(c matcher.Clock) *Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clock = c
	return r
}

// Helper counts one helper-frame mark. A Recorder has no stack to
// attribute a failure to, so it counts the calls and does nothing else.
//
// # Allocation contract
//
// Helper allocates nothing.
func (r *Recorder) Helper() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.helpers++
}

// Fatalf records a failure. The Recorder keeps the first message and
// drops later ones, and each marks the Recorder failed.
//
// Under [Recorder.WithGoexit] it does not return.
//
// # Allocation contract
//
// The first call formats its message. A later call allocates nothing.
func (r *Recorder) Fatalf(format string, args ...any) {
	r.mu.Lock()
	if !r.fatal {
		r.fatal = true
		r.msg = fmt.Sprintf(format, args...)
	}
	r.failed = true
	goexit := r.goexit
	r.mu.Unlock()

	if goexit {
		runtime.Goexit()
	}
}

// Errorf records a failure and returns. The Recorder keeps every
// message, which [Recorder.Messages] returns.
//
// # Allocation contract
//
// A call with a format and no arguments allocates once on average: the
// message. The list of messages grows with each call.
func (r *Recorder) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failed = true
	msg := fmt.Sprintf(format, args...)
	r.errors = append(r.errors, msg)
	if !r.fatal && r.msg == "" {
		r.msg = msg
	}
}

// Failed reports whether any failure arrived, through either
// [Recorder.Fatalf] or [Recorder.Errorf].
//
// # Allocation contract
//
// Failed allocates nothing.
func (r *Recorder) Failed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.failed
}

// Message returns the first message passed to [Recorder.Fatalf], or
// the first passed to [Recorder.Errorf] when nothing fatal arrived. It
// is empty when nothing failed.
//
// # Allocation contract
//
// Message allocates nothing.
func (r *Recorder) Message() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.msg
}

// Messages returns every message passed to [Recorder.Errorf], in call
// order. The slice is a fresh copy, so a caller may keep or sort it
// while the Recorder keeps recording.
//
// # Allocation contract
//
// Messages allocates once: the copy.
func (r *Recorder) Messages() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.errors...)
}

// HelperCalls returns how many times [Recorder.Helper] was called.
//
// # Allocation contract
//
// HelperCalls allocates nothing.
func (r *Recorder) HelperCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.helpers
}
