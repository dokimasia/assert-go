// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

// Package conformance holds this library to the standard.
//
// The standard is language-neutral and lives in its own repository;
// this package is the Go half of proving this implementation matches
// it. Every language repository carries its own equivalent, because
// the check reads the library under test and nothing can do that
// across a language boundary.
//
// # What is checked here
//
// The standard requires two surfaces carrying the same assertions
// under the same names, one that stops the test and one that does not.
// [Surface] reads both and [Members] reports what each exports, so a
// member added to one and forgotten in the other fails the build.
//
// # Dependency position
//
// Imports go/ast, go/parser, go/token, os, path/filepath, slices and
// strings, and nothing outside the standard library. It reads the
// surfaces as source rather than importing them, so it cannot be
// satisfied by a list that has gone stale, and a check that only this
// repository runs adds no dependency to anything that imports the
// library.
package conformance
