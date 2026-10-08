// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package pattern

import "errors"

// ErrOutside reports a pattern outside the portable subset.
var ErrOutside = errors.New("pattern: outside the portable subset")

// Parse returns the pieces of text, for text of the portable subset. The
// anchors that may open and close a pattern leave no piece, because every
// reader of a pattern reads it as a whole. It returns an error that wraps
// [ErrOutside] for text outside the subset, or that is not UTF-8.
func Parse(text string) (Node, error) {
	p, err := newParser(text)
	if err != nil {
		return nil, err
	}
	return p.pattern()
}
