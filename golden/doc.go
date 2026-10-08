// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package golden compares output against a file recording what it
// should be.
//
// Use it where the expected value is too large to write in the test
// and too structured to summarise: a rendered template, a serialised
// document, a generator's output. The file is the assertion, and a
// diff against it is the failure.
//
// [MatchTree] compares a tree of files, such as the directory that a
// generator writes, with a golden directory in the same way.
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
// it records whatever the code does now, and then asserts nothing.
//
// # Scrubbing
//
// Output that contains a timestamp, a digest or a generated identifier
// differs on every run and can never match. A [Scrubber] replaces
// those before the comparison, so the parts that should be stable are
// the parts compared. See [ScrubTimestamps], [ScrubHashes],
// [ScrubRunIDs] and [ScrubJSONFields].
//
// # Failure semantics
//
// Every assertion here stops the test. A test that continued past a
// golden mismatch would report failures about data it already knows is
// wrong.
//
// A failure is a record of the assertion that failed, golden-match,
// golden-match-at or golden-match-json-field, with the golden content as
// want and the output as got, both scrubbed. A missing file or field
// states want as nil. The sentence is the diff that an Equal failure
// shows, so a golden failure reads like every other failure in this
// module: the contract first, then what differed. A failure of
// golden-match-tree states the trees of the entries that differ, as the
// comparisons of package files state them.
//
// A golden file that cannot be read or written, a golden JSON file that
// is no object, a value that is no JSON, and a tree that cannot be read
// or written end the call with a fault, whose call record states the
// verdict error.
//
// # Allocation contracts
//
// The allocation contract of each function counts one call on a seat that
// writes no call record, such as a test's seat while recording is off. A
// failing call allocates its record and its text as well, and a recorded
// call allocates its call record.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert, its internal fault, filetree and matcher,
// and the standard library's bytes, cmp, encoding/json, errors, flag, fmt,
// io, io/fs, os, path/filepath, regexp, runtime, strconv, strings, sync and
// unicode/utf8.
package golden
