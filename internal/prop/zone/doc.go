// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package zone contains the zone list of the definition and every offset
// change of its zones from 1900 until 2100. The definition has both in
// zones.json. The zoned shapes sample the list. In one case of four, they
// generate a value near one of the changes of a zone.
//
// The list keeps the order of the definition, with UTC first, so a zone
// shrinks to UTC. zones.json is a copy of the file of the vendored
// definition. spec sync writes the copy, and a test compares its digest
// with the digest in the vendored manifest. [Locations] resolves each zone
// of the list to its location in a time-zone database.
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
