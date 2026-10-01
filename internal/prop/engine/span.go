// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine

// Span is the range of choices [Start, End) that one generator, element or
// entry of a case made, with its label and its place in the nesting of
// spans.
type Span struct {
	// Label is the id of the generator, or the label of an element or an
	// entry of a collection.
	Label string
	// Start is the index of the span's first choice.
	Start int
	// End is the index after the span's last choice. It equals Start for a
	// span that made no choice.
	End int
	// Depth is the number of spans open around the span.
	Depth int
	// Parent is the index of the innermost span around the span, and -1
	// for a span at the top.
	Parent int
}
