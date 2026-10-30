// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package matcher

import (
	"encoding/json"
	"strings"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/record"
)

// kindPrefix is the prefix of every kind of record. The tools of dokimasia
// reserve it.
const kindPrefix = "dokimi."

// WriteRecord writes text into the output of the test of seat through
// [record.Write]. text is the JSON text of one record of kind. A kind that
// does not match dokimi\.[a-z][a-z0-9-]*, the kind of the call records and a
// text that is not one JSON object end the call of op with a fault through
// [End]. WriteRecord does not write such a record.
//
// # Allocation contract
//
// WriteRecord allocates a copy of text to check it. On a test's seat it also
// allocates the key of the record and what the method Attr allocates. For a
// refused kind or text it allocates the fault.
func WriteRecord(seat Seat, op, kind, text string) {
	seat.Helper()
	if err := recordFault(kind, text); err != nil {
		End(seat, fault.In(op, err))
		return
	}
	record.Write(seat, kind, text)
}

// recordFault returns the fault of a kind or a text that WriteRecord refuses,
// and nil for a kind and a text that it writes.
func recordFault(kind, text string) error {
	switch {
	case !isKind(kind):
		return fault.At(fault.New(`%q does not match dokimi\.[a-z][a-z0-9-]*`, kind), fault.Field("kind"))
	case kind == record.CallKind:
		return fault.At(fault.New("%s is the kind of the call records of assert", kind), fault.Field("kind"))
	case !isObject(text):
		return fault.At(fault.New("the record is not one JSON object"), fault.Field("record"))
	}
	return nil
}

// isKind reports whether kind matches dokimi\.[a-z][a-z0-9-]*.
func isKind(kind string) bool {
	rest, ok := strings.CutPrefix(kind, kindPrefix)
	if !ok || rest == "" || rest[0] < 'a' || rest[0] > 'z' {
		return false
	}
	for i := 1; i < len(rest); i++ {
		if c := rest[i]; (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}
	return true
}

// isObject reports whether text is the JSON of one object.
func isObject(text string) bool {
	return strings.HasPrefix(strings.TrimLeft(text, " \t\r\n"), "{") && json.Valid([]byte(text))
}
