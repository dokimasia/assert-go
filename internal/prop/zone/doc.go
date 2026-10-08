// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package zone contains the definition's zone list and every offset change of
// its zones from 1900 until 2100, as the definition's zones.json states them.
// The zoned shapes sample the list. In one case of four, they generate a
// value near one of a zone's changes.
//
// The list's order is the definition's, with UTC first, so a zone shrinks to
// UTC. zones.json is a copy of the vendored definition's file, which make
// spec-sync writes. A test compares its digest with the vendored manifest's.
// [Locations] resolves each zone of the list to its location in a time-zone
// database.
//
// # Errors
//
// [Locations] returns a fault at the name of the first zone that the
// database lacks.
//
// # Dependency position
//
// Imports the fault package of this module, and the standard library.
package zone
