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

// The text writer takes the sentences of the records of linearizable,
// serializable and snapshot-isolation from this package, which registers
// them while it initialises.
func init() {
	matcher.RegisterSentence(sentence, linearizableID)
	matcher.RegisterSentence(isolationSentence, serializableID, snapshotIsolationID)
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

// isolationSentence returns the sentence of a failing isolation check's
// record f, which the text writer sends to a seat without a Report method.
// It states the contract and the anomaly, then the kinds, and then a line
// for each entry of the explanation.
func isolationSentence(f assert.Failure) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %v\n    kinds: ", f.Contract, f.Detail[anomalyField])
	kinds, _ := f.Detail[kindsField].([]Anomaly)
	for i, k := range kinds {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(k.String())
	}
	evidence, _ := f.Detail[explanationField].([]Evidence)
	for _, e := range evidence {
		b.WriteString("\n    ")
		switch e := e.(type) {
		case Edge:
			writeEdge(&b, e)
		case Observation:
			writeObservation(&b, e)
		}
	}
	return b.String()
}

// writeEdge writes the text of an edge of a cycle: its two calls and its
// relation, and the key and the values that prove it.
func writeEdge(b *strings.Builder, e Edge) {
	fmt.Fprintf(b, "call %d -%v-> call %d: ", e.From, e.Relation, e.To)
	if e.Relation == WW {
		text.Fprintf(b, "call %d appended %#v to %#v after %#v", e.To, e.Next, e.Key, e.Value)
		return
	}
	if e.Relation == WR {
		text.Fprintf(b, "call %d read %#v ending in %#v", e.To, e.Key, e.Value)
		return
	}
	if e.Empty {
		text.Fprintf(b, "call %d read [] from %#v, and call %d appended %#v to it", e.From, e.Key, e.To, e.Next)
		return
	}
	text.Fprintf(b, "call %d read %#v ending in %#v, and call %d appended %#v after it",
		e.From, e.Key, e.Value, e.To, e.Next)
}

// writeObservation writes the text of the observation of an anomaly that is
// no cycle: the reads and the appends that show it.
func writeObservation(b *strings.Builder, o Observation) {
	switch o.Anomaly {
	case GarbageRead:
		text.Fprintf(b, "call %d read %s from %#v, and no transaction appended %#v",
			o.Calls[0], listText(o.Reads[0]), o.Key, o.Value)
	case DuplicateAppend:
		text.Fprintf(
			b,
			"call %d read %s from %#v, which contains %#v twice",
			o.Calls[0],
			listText(o.Reads[0]),
			o.Key,
			o.Value,
		)
	case InternalInconsistency:
		text.Fprintf(b, "call %d read %s from %#v", o.Calls[0], listText(o.Reads[0]), o.Key)
		if o.Whole {
			fmt.Fprintf(b, ", and it knew the list was %s", listText(o.Expected))
		} else if len(o.Expected) > 0 {
			fmt.Fprintf(b, ", and it knew the list ended with %s", listText(o.Expected))
		}
		if o.HasFuture {
			text.Fprintf(b, ", and %#v is a value that call %d appends later", o.Future, o.Calls[0])
		}
	case IncompatibleOrder:
		text.Fprintf(b, "calls %d and %d read %s and %s from %#v, and neither is a prefix of the other",
			o.Calls[0], o.Calls[1], listText(o.Reads[0]), listText(o.Reads[1]), o.Key)
	case AbortedRead:
		text.Fprintf(
			b,
			"call %d read %#v from %#v, which call %d appended and aborted",
			o.Calls[0],
			o.Value,
			o.Key,
			o.Appender,
		)
	default:
		text.Fprintf(b, "call %d read %#v ending in %#v, which call %d followed with %#v",
			o.Calls[0], o.Key, o.Value, o.Appender, o.Next)
	}
}

// listText returns the text of a list of values, each as a Go literal.
func listText(values []any) string {
	var b strings.Builder
	b.WriteString("[")
	for i, v := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		text.Fprintf(&b, "%#v", v)
	}
	b.WriteString("]")
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
