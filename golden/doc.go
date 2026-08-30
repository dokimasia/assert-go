// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package golden compares output against a file recording what it
// should be.
//
// Use it where the expected value is too large to write in the test
// and too structured to summarise: a rendered template, a serialised
// document, a generator's output. The file is the assertion, and a
// diff against it is the failure.
//
// # Updating
//
// A golden file is rewritten by passing [ShouldUpdate] as the update
// argument and running the test with -update:
//
//	golden.Match(t, "response.json", got, golden.ShouldUpdate())
//
//	go test ./... -update
//
// Read the diff before updating. A golden file updated without reading
// it records whatever the code now does, which is the opposite of an
// assertion.
//
// # Scrubbing
//
// Output holding a timestamp, a digest or a generated identifier
// differs on every run and can never match. A [Scrubber] replaces
// those before the comparison, so the parts that should be stable are
// the parts compared. See [ScrubTimestamps], [ScrubHashes],
// [ScrubRunIDs] and [ScrubJSONFields].
//
// # Failure semantics
//
// Every assertion here stops the test. A test that carried on past a
// golden mismatch would report failures about data it already knows is
// wrong.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert, and the standard library's
// encoding/json, flag, fmt, os, path/filepath, regexp and strings.
//
// The comparison is [go.dokimi.dev/assert.Equal] rather than a diff of
// its own, so a golden failure reads like every other failure in this
// module: the contract first, then what differed.
package golden
