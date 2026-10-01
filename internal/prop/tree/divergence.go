// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package tree

import (
	"errors"
	"strconv"

	"go.dokimi.dev/assert/internal/prop/choice"
)

// ErrRepeated reports a case whose choices repeat a tested case. It ends
// the case, which does not count.
var ErrRepeated = errors.New("tree: the case repeats a tested case")

// DivergenceError reports a body that made other requests after the same
// values than an earlier case made. A [Walker] returns it as the error of
// the step or the end where the two cases part.
type DivergenceError struct {
	// Index is the position of the choice where the cases part.
	Index int
	// Recorded are the bounds that the tree recorded at the position, and
	// nil where an earlier case ended.
	Recorded *choice.Bounds
	// Requested are the bounds that this case requested at the position,
	// and nil where it ended.
	Requested *choice.Bounds
}

// Error returns the position where the cases part, and which of them
// ended there.
func (d *DivergenceError) Error() string {
	text := "tree: the body diverged at choice " + strconv.Itoa(d.Index)
	if d.Recorded == nil {
		return text + ", where an earlier case ended"
	}
	if d.Requested == nil {
		return text + ", where this case ended"
	}
	return text + ", where it requested other bounds"
}
