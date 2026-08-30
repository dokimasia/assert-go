// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golden

import (
	"fmt"
	"regexp"
	"strings"
)

// What a scrubber writes in place of what it removed. The text is
// visible in the golden file, so a reader can see that a value was
// replaced rather than wondering why it looks wrong.
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
func ScrubTimestamps() Scrubber {
	return func(s string) string {
		return timestampPattern.ReplaceAllString(s, scrubbedTimestamp)
	}
}

// ScrubHashes replaces hex digests between 32 and 128 characters,
// which covers MD5 through SHA-512.
func ScrubHashes() Scrubber {
	return func(s string) string {
		return hashPattern.ReplaceAllString(s, scrubbedHash)
	}
}

// ScrubRunIDs replaces identifiers of the form run_ followed by
// sixteen lowercase alphanumerics.
func ScrubRunIDs() Scrubber {
	return func(s string) string {
		return runIDPattern.ReplaceAllString(s, scrubbedRunID)
	}
}

// ScrubJSONFields replaces the value of each named JSON field.
//
//	golden.ScrubJSONFields("created_at", "token")
//
// It matches the field's text rather than parsing the document, so it
// works on output that is nearly JSON as well as output that is. The
// cost is that a field name appearing inside a string value is
// replaced too; name fields that will not collide.
func ScrubJSONFields(fields ...string) Scrubber {
	if len(fields) == 0 {
		return func(s string) string { return s }
	}

	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = regexp.QuoteMeta(f)
	}
	pattern := regexp.MustCompile(
		fmt.Sprintf(`("(?:%s)"\s*:\s*)"[^"]*"`, strings.Join(quoted, "|")),
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
