// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"fmt"
	"reflect"
	"strings"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/text"
)

// The text writer takes the sentence of the record of linearizable from this
// package, which registers it while it initialises.
func init() {
	matcher.RegisterSentence(sentence, linearizableID)
}

// sentence returns the sentence of a failing check's record f, which the
// text writer sends to a seat without a Report method. It states the
// contract, the outcome with the reported partition and the limit, and then
// the counts, the frontier's order of calls, its states and the calls that
// the model rejected there, each on a line of its own.
func sentence(f assert.Failure) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %v in the partition of %s", f.Contract, f.Detail[outcomeField],
		keysText(f.Detail[partitionField]))
	if limit, ok := f.Detail[limitField].(Limit); ok {
		fmt.Fprintf(&b, ", at the %v limit", limit)
	}
	fmt.Fprintf(&b, "\n    steps %v, partitions %v, calls %v, concurrency %v", f.Detail[stepsField],
		f.Detail[partitionsField], f.Detail[callsField], f.Detail[concurrencyField])
	writeSpans(&b, "linearized", f.Detail[linearizedField])
	b.WriteString("\n    states: ")
	if states := reflect.ValueOf(f.Detail[statesField]); states.Kind() == reflect.Slice {
		for i := range states.Len() {
			if i > 0 {
				b.WriteString(", ")
			}
			text.Fprintf(&b, "%#v", states.Index(i).Interface())
		}
	}
	writeSpans(&b, "rejected", f.Detail[candidatesField])
	return b.String()
}

// keysText returns the text of the keys of a partition, each as a Go
// literal, and "every key" for a partition of every key.
func keysText(keys any) string {
	list, _ := keys.([]any)
	if len(list) == 0 {
		return "every key"
	}
	var b strings.Builder
	for i, key := range list {
		if i > 0 {
			b.WriteString(", ")
		}
		text.Fprintf(&b, "%#v", key)
	}
	return b.String()
}

// writeSpans writes a line of the label and the text of each span of spans,
// a []Span, or "none" for no span. A span's text states its call, its
// operation with its arguments as Go literals, and its output, or that its
// outcome is unknown.
func writeSpans(b *strings.Builder, label string, spans any) {
	list, _ := spans.([]Span)
	b.WriteString("\n    " + label + ": ")
	if len(list) == 0 {
		b.WriteString("none")
	}
	for i, s := range list {
		if i > 0 {
			b.WriteString("; ")
		}
		fmt.Fprintf(b, "call %d %s(", s.Call, s.Op.Operation)
		for j, arg := range s.Op.Args {
			if j > 0 {
				b.WriteString(", ")
			}
			text.Fprintf(b, "%#v", arg)
		}
		b.WriteString(")")
		if s.Op.Known {
			text.Fprintf(b, " → %#v", s.Op.Output)
		} else {
			b.WriteString(", outcome unknown")
		}
	}
}
