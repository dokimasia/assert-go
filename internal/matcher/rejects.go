// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"runtime"
	"sync"

	"go.dokimi.dev/assert/internal/record"
)

// rejection is the seat of the check that [Rejects] drives. It keeps each
// record of the check in call order, whether the check failed at all, and
// the fault that ended the check. Its Calls keep the records of the check's
// calls, for the call of Rejects. Its context derives from the context of
// the seat of Rejects, and ends with the check.
//
// Fatalf, an aborting record and a fault that ends its call end the calling
// goroutine once they are kept, as a test's Fatalf ends a test, so the check
// stops at its first failure that stops. The check runs on a goroutine of
// its own for that reason.
//
// It is not the public recorder, because the package that declares the
// recorder imports this one.
type rejection struct {
	calls
	bodyContext

	mu      sync.Mutex
	failed  bool
	records []Failure
	// fault is the fault that ended the check, and nil for a check that no
	// fault ended.
	fault error
}

func (*rejection) Helper() {}

// Fatalf marks the check failed and ends it.
func (r *rejection) Fatalf(string, ...any) {
	r.mark()
	runtime.Goexit()
}

// Errorf marks the check failed.
func (r *rejection) Errorf(string, ...any) { r.mark() }

// Report keeps f, and ends the check when f is aborting.
func (r *rejection) Report(f Failure, aborting bool) {
	r.mu.Lock()
	r.failed = true
	r.records = append(r.records, f)
	r.mu.Unlock()

	if aborting {
		runtime.Goexit()
	}
}

// ReportFault keeps a fault that ends its call and ends the check, as a
// failure that stops ends it. It passes a noted fault to the seat of
// Rejects, as [NoteFault] notes it there, and the check runs on.
func (r *rejection) ReportFault(err error, ending bool) {
	if !ending {
		NoteFault(r.parent, err)
		return
	}
	r.mu.Lock()
	r.fault = err
	r.mu.Unlock()

	runtime.Goexit()
}

// mark marks the check failed.
func (r *rejection) mark() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.failed = true
}

// outcome returns a copy of the check's records, whether it failed, and the
// fault that ended it.
func (r *rejection) outcome() (records []Failure, failed bool, fault error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]Failure(nil), r.records...), r.failed, r.fault
}

// Rejects runs fn against an implementation that fn is meant to reject,
// and reports when fn passes. It returns the failure records of fn, in call
// order, and none when fn reported none. Its failure is a record of
// rejects without detail, whose contract is msg.
//
// fn receives a seat of its own, which keeps every record of fn and ends
// fn at its first failure that stops, as a test's Fatalf ends a test. A
// message that fn passes to Fatalf or Errorf of that seat fails fn without a
// record. A fault that ends fn ends the call of Rejects with the same fault,
// through [Running.Fault], and Rejects returns no record: a check that ends
// with a fault has no verdict on the implementation. A fault that fn notes
// passes to seat as [NoteFault] notes it. The calls of fn are recorded under
// the call of Rejects, as its run 1. The seat's context derives from the
// context of seat, as [ContextOf] reads it, and Rejects cancels it when fn
// ends.
//
//	got := matcher.Rejects(seat, matcher.Fatal, "an unbounded pool fails the check",
//	    func(s matcher.Seat) { handsOutEveryItem(s, unboundedPool{}) })
//
// # Concurrency
//
// fn runs on a goroutine of its own, and Rejects returns once that
// goroutine has ended. A panic in fn is not recovered: it stops the
// process.
//
// # Allocation contract
//
// A call whose check fails at one aborting assertion without detail
// allocates 7 times, the seat and the goroutine of the check, the check's
// failure and the returned copy of its records included.
func Rejects(seat Seat, mode Mode, msg string, fn func(Seat)) []Failure {
	seat.Helper()

	run := Begin(seat)
	r := &rejection{parent: seat}
	record.Run(&r.calls, run.Slot(), nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(r)
	}()
	<-done
	r.end()
	run.Slot().Take(&r.calls, record.NoPhase)

	records, failed, fault := r.outcome()
	if fault != nil {
		run.Fault(mode, "rejects", msg, fault)
		return nil
	}
	if !failed {
		run.Fail(mode, "rejects", msg, nil)
		return records
	}
	run.Pass(mode, "rejects", msg)
	return records
}
