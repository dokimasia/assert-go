// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package filetree

import (
	"fmt"
	"slices"
	"strings"

	"go.dokimi.dev/assert/internal/matcher"
)

// Sentence returns the sentence of the record f of a comparison of trees,
// which the text writer sends to a seat without a Report method. It states
// the contract and the number of paths that differ, then a line of each path
// that the record lists, with the wanted entry after - and the entry read
// after +, as the Go expressions of package files that state them, and last
// the number of the paths that the record does not list. A record without a
// wanted tree states that the golden tree is missing.
//
// # Allocation contract
//
// Sentence allocates the text, the paths of the record, and the text of each
// entry.
func Sentence(f matcher.Failure) string {
	want, _ := f.Detail[WantField].(Tree)
	got, _ := f.Detail[GotField].(Tree)
	differences, _ := f.Detail[DifferencesField].(int)

	var b strings.Builder
	switch {
	case f.Detail[WantField] == nil:
		fmt.Fprintf(&b, "%s: the golden tree is missing, and the output has %d entries (+got)", f.Contract, differences)
	case differences == 1:
		fmt.Fprintf(&b, "%s: 1 path differs (-want +got)", f.Contract)
	default:
		fmt.Fprintf(&b, "%s: %d paths differ (-want +got)", f.Contract, differences)
	}
	paths := slices.Concat(want.Paths(), got.Paths())
	slices.Sort(paths)
	paths = slices.Compact(paths)
	for _, path := range paths {
		b.WriteString("\n\t")
		b.WriteString(path)
		b.WriteByte(':')
		if w, ok := want[path]; ok {
			b.WriteString(" -")
			b.WriteString(w.GoString())
		}
		if g, ok := got[path]; ok {
			b.WriteString(" +")
			b.WriteString(g.GoString())
		}
	}
	if more := differences - len(paths); more > 0 {
		fmt.Fprintf(&b, "\n\t… and %d more", more)
	}
	return b.String()
}
