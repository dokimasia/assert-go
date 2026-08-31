// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matchertest

import (
	"fmt"
	"sync"

	"go.dokimi.dev/assert/internal/matcher"
)

// Seat records what a matcher reported. It satisfies the seat
// interface every matcher takes, without depending on the package that
// declares it: the method set is the contract, not the name.
//
// The zero value is usable. Every method is safe for concurrent use,
// so a matcher that reports from a goroutine can be tested.
type Seat struct {
	mu      sync.Mutex
	fatals  []string
	errs    []string
	records []matcher.Failure
	helpers int
}

// Report records one failure as the record it is, and renders it
// through the matching string path so the message checks still see
// what a real seat would.
//
// Satisfying this interface is what lets a case state the fields an
// assertion reports rather than words its sentence happens to hold.
func (s *Seat) Report(f matcher.Failure, aborting bool) {
	s.mu.Lock()
	s.records = append(s.records, f)
	s.mu.Unlock()

	if aborting {
		s.Fatalf("%s", matcher.Render(f))
		return
	}
	s.Errorf("%s", matcher.Render(f))
}

// Records returns a copy of every record reported, in call order.
func (s *Seat) Records() []matcher.Failure {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]matcher.Failure(nil), s.records...)
}

// Helper counts one helper-frame mark.
func (s *Seat) Helper() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.helpers++
}

// Fatalf records a failure reported through the aborting path. It
// returns, unlike a real seat, so a test can assert on what a matcher
// reported and then carry on.
func (s *Seat) Fatalf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fatals = append(s.fatals, fmt.Sprintf(format, args...))
}

// Errorf records a failure reported through the recording path.
func (s *Seat) Errorf(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.errs = append(s.errs, fmt.Sprintf(format, args...))
}

// Fatals returns a copy of every message reported through Fatalf, in
// call order.
func (s *Seat) Fatals() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.fatals...)
}

// Errs returns a copy of every message reported through Errorf, in
// call order.
func (s *Seat) Errs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.errs...)
}

// HelperCalls returns how many times [Seat.Helper] was called.
func (s *Seat) HelperCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.helpers
}

// Failed reports whether anything was recorded through either path.
func (s *Seat) Failed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.fatals) > 0 || len(s.errs) > 0
}

// First returns the first message recorded through either path,
// preferring the aborting one. It is empty when nothing was recorded.
//
// Most matcher tests assert on one failure, and reaching for this
// instead of indexing a slice keeps them from panicking when the
// matcher under test wrongly reported nothing.
func (s *Seat) First() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.fatals) > 0 {
		return s.fatals[0]
	}
	if len(s.errs) > 0 {
		return s.errs[0]
	}
	return ""
}
