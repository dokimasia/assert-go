// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert

import (
	"fmt"
	"runtime"
	"sync"
)

// Recorder is a [TB] that records a failure instead of stopping the
// test, so an assertion can be tested by reading what it reported.
//
// The zero value is not usable; call [NewRecorder]. Safe for
// concurrent use.
type Recorder struct {
	mu      sync.Mutex
	goexit  bool
	failed  bool
	fatal   bool
	msg     string
	errors  []string
	helpers int
}

// NewRecorder returns a Recorder ready for use.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// WithGoexit makes Fatalf call [runtime.Goexit] after recording, which
// stops a body running past an assertion it already failed. Returns the
// receiver.
func (r *Recorder) WithGoexit() *Recorder {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.goexit = true
	return r
}

// Helper counts one helper-frame mark.
func (r *Recorder) Helper() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.helpers++
}

// Fatalf records a failure. The first message is kept and later ones
// are dropped. Calls [runtime.Goexit] when configured by
// [Recorder.WithGoexit].
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

// Errorf records a failure and returns.
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

// Failed reports whether any failure was recorded.
func (r *Recorder) Failed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.failed
}

// Msg returns the first message passed to [Recorder.Fatalf], or the
// first passed to [Recorder.Errorf] when nothing was fatal. Empty when
// nothing failed.
func (r *Recorder) Msg() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.msg
}

// Errors returns a copy of every message passed to [Recorder.Errorf],
// in call order.
func (r *Recorder) Errors() []string {
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
