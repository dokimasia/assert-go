// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

// Call is an open call of a [History], through which its client records the
// completion. It is a value of two words, and two copies of one Call record
// the completion of one call. The zero Call is no call.
type Call struct {
	// history is the history of the call.
	history *History
	// index is the index of the call's invocation.
	index int
}

// OK records that the call returned output and took effect.
//
// # Allocation contract
//
// OK allocates the growth of the history's events.
//
// # Panics
//
// OK panics when the call has completed already.
func (c Call) OK(output any) {
	c.history.complete(c.index, OK, output, nil)
}

// Fail records that the call returned err, took no effect, and returned
// nothing that a spec checks, such as a refused connection. An error that
// states what the subject observed is an output for [Call.OK]: a
// compare-and-set that refuses because the value differs returns false.
//
// # Allocation contract
//
// Fail allocates the growth of the history's events.
//
// # Panics
//
// Fail panics when the call has completed already.
func (c Call) Fail(err error) {
	c.history.complete(c.index, Fail, nil, err)
}

// Unknown records that the call ended without an outcome, such as a
// timeout, a lost reply or a crash. The call may still take effect, so its
// client's next invocation starts on a new process.
//
// # Allocation contract
//
// Unknown allocates the growth of the history's events.
//
// # Panics
//
// Unknown panics when the call has completed already.
func (c Call) Unknown(err error) {
	c.history.complete(c.index, Unknown, nil, err)
}
