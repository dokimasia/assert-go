// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"fmt"
	"regexp"
	"strings"
)

// What a scrubber writes in place of what it removed. The text is
// visible in the golden file, so a reader sees that a value was
// replaced.
const (
	scrubbedTimestamp = "SCRUBBED_TIMESTAMP"
	scrubbedHash      = "SCRUBBED_HASH"
	scrubbedRunID     = "SCRUBBED_RUN_ID"
	scrubbedValue     = "SCRUBBED"
)

// Scrubber replaces content that differs between runs, so a
// comparison sees only the parts that should be stable.
//
// A Scrubber is applied to the value under test before it is compared,
// and to the golden file's contents before they are diffed, so both
// sides are scrubbed the same way.
type Scrubber func(string) string

// patterns the built-in scrubbers match.
var (
	// timestampPattern matches ISO-8601 and RFC-3339, with or without
	// fractional seconds and with either a zone offset or Z.
	timestampPattern = regexp.MustCompile(
		`\d{4}-\d{2}-\d{2}[Tt ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[Zz]|[+-]\d{2}:?\d{2})?`,
	)
	// hashPattern matches a hex digest from MD5 through SHA-512.
	hashPattern = regexp.MustCompile(`\b[0-9a-fA-F]{32,128}\b`)
	// runIDPattern matches the run_ identifiers a scheduler hands out.
	runIDPattern = regexp.MustCompile(`\brun_[0-9a-z]{16}\b`)
)

// ScrubTimestamps replaces ISO-8601 and RFC-3339 timestamps.
//
// # Allocation contract
//
// ScrubTimestamps allocates nothing: the scrubber has no state.
func ScrubTimestamps() Scrubber {
	return func(s string) string {
		return timestampPattern.ReplaceAllString(s, scrubbedTimestamp)
	}
}

// ScrubHashes replaces hex digests between 32 and 128 characters,
// which covers MD5 through SHA-512.
//
// # Allocation contract
//
// ScrubHashes allocates nothing: the scrubber has no state.
func ScrubHashes() Scrubber {
	return func(s string) string {
		return hashPattern.ReplaceAllString(s, scrubbedHash)
	}
}

// ScrubRunIDs replaces identifiers of the form run_ followed by
// sixteen lowercase alphanumerics.
//
// # Allocation contract
//
// ScrubRunIDs allocates nothing: the scrubber has no state.
func ScrubRunIDs() Scrubber {
	return func(s string) string {
		return runIDPattern.ReplaceAllString(s, scrubbedRunID)
	}
}

// jsonScalar matches the text of a JSON value that is no object and no
// array: a string with its escapes, a number, true, false or null.
const jsonScalar = `"(?:[^"\\]|\\.)*"|-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?|(?:true|false|null)\b`

// ScrubJSONFields replaces the value of each named JSON field with the
// string "SCRUBBED": a string, a number, true, false or null.
//
//	golden.ScrubJSONFields("created_at", "token", "seconds")
//
// It matches the text of a field and does not parse the document, so it
// works on output that is nearly JSON, and it leaves an object or an array
// as it is. A field name inside a string value is replaced too, so name
// fields that no string value contains.
//
// # Allocation contract
//
// ScrubJSONFields compiles a regular expression of the fields and of every
// scalar value: 113 allocations for one field. It allocates nothing for no
// field.
func ScrubJSONFields(fields ...string) Scrubber {
	if len(fields) == 0 {
		return func(s string) string { return s }
	}

	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = regexp.QuoteMeta(f)
	}
	pattern := regexp.MustCompile(
		fmt.Sprintf(`("(?:%s)"\s*:\s*)(?:%s)`, strings.Join(quoted, "|"), jsonScalar),
	)

	return func(s string) string {
		return pattern.ReplaceAllString(s, `${1}"`+scrubbedValue+`"`)
	}
}

// scrub applies every scrubber in order.
func scrub(s string, scrubbers []Scrubber) string {
	for _, f := range scrubbers {
		s = f(s)
	}
	return s
}
