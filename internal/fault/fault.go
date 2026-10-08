// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package fault

import (
	"strings"

	"go.dokimi.dev/assert/internal/text"
)

// separator joins the parts of a fault's text.
const separator = ": "

// Error is a fault, an error of this module: the operation that failed,
// where in its input, the kind of its condition, the reason and the cause.
//
// The fields are the contract. Code and tests compare them, and never the
// text that [Error.Error] renders.
type Error struct {
	// Op is the operation that failed: the package and its exported
	// function, as "prop.ShapeOf". It is empty until [In] states it.
	Op string
	// Path is where in the input the fault is, and empty for an input
	// without parts.
	Path Path
	// Kind is the sentinel that errors.Is matches, and nil for a condition
	// that a caller has no reason to test.
	Kind error
	// Reason states what is wrong in one clause, in lowercase and without a
	// final period.
	Reason string
	// Err is the cause, and nil for a fault without one.
	Err error
}

// New returns a fault whose reason is format with args, as fmt.Sprintf
// formats them. A value that contains itself, or that has more than 65,536
// parts, states its bounded text as the package text writes it.
//
// # Allocation contract
//
// New allocates the fault and its reason, and what fmt.Sprintf allocates
// for args: two allocations for a reason without arguments.
func New(format string, args ...any) *Error {
	return &Error{Reason: text.Sprintf(format, args...)}
}

// Of returns a fault of kind whose reason is format with args, as [New]
// formats them.
//
// # Allocation contract
//
// Of allocates what [New] allocates.
func Of(kind error, format string, args ...any) *Error {
	return &Error{Kind: kind, Reason: text.Sprintf(format, args...)}
}

// Because makes err the cause of f, and returns f, so that a fault with a
// cause is one expression: fault.Of(ErrShape, "the text is no
// JSON").Because(err). It allocates nothing.
func (f *Error) Because(err error) *Error {
	f.Err = err
	return f
}

// At returns err at the segments segs, outermost first: a copy of a fault
// with segs in front of its path, or a new fault at segs whose cause is err
// when err is no fault. It leaves the fault that err is unchanged.
//
// # Allocation contract
//
// At allocates the fault it returns and its path: two allocations.
func At(err error, segs ...Segment) error {
	f, ok := err.(*Error) //nolint:errorlint // At extends the fault that err is, and never one that err wraps
	if !ok {
		path := make(Path, len(segs))
		copy(path, segs)
		return &Error{Path: path, Err: err}
	}
	out := *f
	out.Path = make(Path, len(segs)+len(f.Path))
	copy(out.Path, segs)
	copy(out.Path[len(segs):], f.Path)
	return &out
}

// In returns err as a fault of op: a copy of a fault without an operation,
// with op as its operation, and otherwise a new fault of op whose cause is
// err. A fault that names its operation keeps it, so the text of a fault
// that crossed two packages names both operations.
//
// # Allocation contract
//
// In allocates the fault it returns: one allocation.
func In(op string, err error) error {
	f, ok := err.(*Error) //nolint:errorlint // In names the fault that err is, and never one that err wraps
	if !ok || f.Op != "" {
		return &Error{Op: op, Err: err}
	}
	out := *f
	out.Op = op
	return &out
}

// Error returns the operation, the path, the reason and the text of the
// cause, each part that is not empty, joined by a colon and a space. A
// fault without a reason states the text of its kind in place of it.
//
// # Allocation contract
//
// Error allocates the text it returns and the text of its path, and what
// the Error of its cause allocates: two allocations for a fault with a path
// and a cause of a constant text.
func (f *Error) Error() string {
	parts := [...]string{f.Op, f.Path.String(), f.reason(), f.cause()}
	n := 0
	for _, part := range parts {
		if part != "" {
			n += len(separator) + len(part)
		}
	}
	var b strings.Builder
	b.Grow(n)
	for _, part := range parts {
		if part == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(separator)
		}
		b.WriteString(part)
	}
	return b.String()
}

// Unwrap returns the cause of f, so that errors.Is and errors.As see
// through it. It allocates nothing.
func (f *Error) Unwrap() error { return f.Err }

// Is reports whether target is the kind of f, so that errors.Is matches a
// fault by its kind. It allocates nothing.
func (f *Error) Is(target error) bool { return target == f.Kind }

// reason returns the reason of f, or the text of its kind for a fault
// without a reason.
func (f *Error) reason() string {
	if f.Reason == "" && f.Kind != nil {
		return f.Kind.Error()
	}
	return f.Reason
}

// cause returns the text of the cause of f, and the empty string for a
// fault without one.
func (f *Error) cause() string {
	if f.Err == nil {
		return ""
	}
	return f.Err.Error()
}
