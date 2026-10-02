// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

// NoLeaks is the leak check over the stacks that a test's dump writes,
// read into buffers of at most limit bytes.
var NoLeaks = noLeaks

// EndAbandoned ends a subject of CompletesWithin whose deadline passed
// first, with the value that the subject panicked with or nil.
func EndAbandoned(raised any) {
	s := newSubject()
	s.expire()
	s.end(raised)
}
