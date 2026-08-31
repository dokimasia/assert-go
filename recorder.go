// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"fmt"
	"runtime"
	"sync"

	"go.dokimi.dev/assert/internal/matcher"
)

// Recorder is a [TB] that records a failure instead of stopping the
// test, so an assertion can be tested by reading what it reported.
//
// It follows a real test seat in the one way that matters for reading
// a failure back: the first fatal message is the one kept, because in
// a real test nothing after that call runs. Non-fatal failures
// accumulate instead, matching a seat whose test carries on.
//
// The zero value is not usable; call [NewRecorder]. Every method is
// safe for concurrent use, so a driven body may run on a goroutine of
// its own.
type Recorder struct {
	mu sync.Mutex
	// goexit makes Fatalf end the calling goroutine once it has
	// recorded, so a driven body stops where a real test would.
	goexit bool
	// failed records that some failure arrived, fatal or not.
	failed bool
	// fatal records that a fatal failure arrived, which fixes msg
	// against every later call.
	fatal bool
	// msg is the first fatal message, or the first non-fatal one when
	// nothing fatal has arrived.
	msg string
	// errors holds every non-fatal message in call order.
	errors []string
	// records holds every failure that arrived as a record, in call
	// order, whether fatal or not. An assertion reports one; a
	// message passed straight to Fatalf or Errorf leaves none.
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
func NewRecorder() *Recorder {
	return &Recorder{}
}

// WithGoexit makes [Recorder.Fatalf] call [runtime.Goexit] once it has
// recorded, and returns the receiver so the call chains onto
// [NewRecorder].
//
// This is what stops a driven body running past an assertion it
// already failed. Goexit ends the calling goroutine, so a body
// configured this way must run on a goroutine of its own.
func (r *Recorder) WithGoexit() *Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.goexit = true
	return r
}

// Report records one failure and, when it is fatal, stops the driven
// body the way [Recorder.Fatalf] does.
//
// This is what makes a Recorder a [Reporter]: an assertion hands it
// the record, so a test can read the assertion's own fields rather
// than search its sentence for words. The rendered sentence is kept
// too, so [Recorder.Message] answers what it always did.
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

// Failures answers every record that arrived, in call order. The
// slice is a fresh copy, so a caller may hold or sort it while the
// Recorder keeps recording.
func (r *Recorder) Failures() []Failure {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]Failure(nil), r.records...)
}

// Clock answers the clock this Recorder hands assertions, which is the
// platform clock unless [Recorder.WithClock] set another.
func (r *Recorder) Clock() matcher.Clock {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.clock == nil {
		return matcher.System{}
	}
	return r.clock
}

// WithClock makes assertions reported through this Recorder read c
// rather than the platform clock, and returns the receiver so the call
// chains onto [NewRecorder].
func (r *Recorder) WithClock(c matcher.Clock) *Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clock = c
	return r
}

// Helper counts one helper-frame mark. A Recorder has no stack to
// attribute a failure to, so counting is all it does.
func (r *Recorder) Helper() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.helpers++
}

// Fatalf records a failure. The first message is kept and later ones
// are dropped, though each still marks the Recorder failed.
//
// Under [Recorder.WithGoexit] this does not return.
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

// Errorf records a failure and returns. Every message is kept; read
// them with [Recorder.Errors].
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
func (r *Recorder) Failed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.failed
}

// Message returns the first message passed to [Recorder.Fatalf], or
// the first passed to [Recorder.Errorf] when nothing fatal arrived. It
// is empty when nothing failed.
func (r *Recorder) Message() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.msg
}

// Messages returns every message passed to [Recorder.Errorf], in call
// order. The slice is a fresh copy, so a caller may hold or sort it
// while the Recorder keeps recording.
func (r *Recorder) Messages() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.errors...)
}

// HelperCalls returns how many times [Recorder.Helper] was called.
func (r *Recorder) HelperCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.helpers
}
