// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package golangci registers the analyzer of go.dokimi.dev/assert/lint with
// golangci-lint, as the module plugin assertlint.
//
// golangci-lint runs a module plugin from a binary that its command custom
// builds. A .custom-gcl.yml specifies the module:
//
//	version: v2.14.0
//	plugins:
//	  - module: go.dokimi.dev/assert/lint/golangci
//	    version: v0.1.0
//
// golangci-lint custom builds the binary custom-gcl from it. A .golangci.yml
// enables the linter:
//
//	version: "2"
//	linters:
//	  enable:
//	    - assertlint
//	  settings:
//	    custom:
//	      assertlint:
//	        type: module
//	issues:
//	  max-issues-per-linter: 0
//	  max-same-issues: 0
//
// The linter takes no settings. By default golangci-lint shows 3 reports of
// one text and 50 reports of one linter, and many reports of assertlint
// share a text, such as "compare: state the check with Equal". The limits
// of 0 show every report. custom-gcl run --fix applies the fixes that
// assertlint -fix applies.
//
// # Dependency position
//
// Imports go.dokimi.dev/assert/lint, register of
// github.com/golangci/plugin-module-register, analysis of
// golang.org/x/tools, and fmt of the standard library.
package golangci
