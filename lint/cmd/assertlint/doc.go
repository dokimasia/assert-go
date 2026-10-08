// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Command assertlint reports the checks that a test writes by hand and that
// an assertion of go.dokimi.dev/assert states, through the analyzer of
// go.dokimi.dev/assert/lint.
//
//	assertlint ./...
//	assertlint -fix ./...
//
// It prints each check, and exits with status 3 when it reports one. Under
// -fix, it applies the fix of each check that has one, prints nothing, and
// exits with status 0. A second run without -fix reports the checks that have
// no fix.
//
// The go command runs it as a vet tool and as a fix tool:
//
//	go vet -vettool=$(command -v assertlint) ./...
//	go vet -fix -vettool=$(command -v assertlint) ./...
//	go fix -fixtool=$(command -v assertlint) ./...
//
// go vet prints each check and exits with status 1. go vet -fix and go fix
// apply the fixes instead.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert/lint, and singlechecker of golang.org/x/tools.
package main
