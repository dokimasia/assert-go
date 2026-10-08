// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package store reads and writes the store of a property: the minimal
// failing case of each of its failures, one JSON entry per file in a
// directory of the test, which the next run tries first.
//
// An [Entry] is the case with what a reviewer needs to read it: the
// property's contract, the failure's [Identity], the replay token of the
// case's choices, and each drawn value as a typed literal. [Entry.Name]
// derives the file's name from the contract and the token, so the same
// failure, shrunk to the same choices, has the same name in every
// language, and two branches that each find a failure add two files.
//
// # Reading
//
// [Read] gives the content of one file a [Verdict] for one property:
//
//   - [Replay]: an entry of [Format] for the property, whose choices the
//     run tries first.
//   - [Other]: an entry of [Format] for another property of the test.
//   - [Skip]: an entry of a later format, or whose token is of a later
//     version. The run notes it and goes on.
//   - [Damaged]: any other file. The test fails, as it fails on a damaged
//     golden file.
//
// The reader is strict, so every language gives a file the same verdict.
// A file is damaged when it is not one JSON object in UTF-8, repeats a
// name within an object, nests objects and arrays more than 64 levels
// deep, lacks a field of its format, states another field, or states a
// field out of its form. A draw's value is left unread, because only the
// choices replay. [Load] reads a whole directory.
//
// # Writing
//
// [Save] writes an entry as indented JSON, and never overwrites a file:
// an entry whose file exists is kept as it is.
//
// # Errors
//
// [Read] and [Load] return faults of the kind [ErrDamaged] for a damaged
// file, and of the kind [ErrLater] for a skipped one. [Save] returns a
// fault of the kind [ErrInvalid] for an entry that [Read] would not
// replay. A fault's path leads through the file's JSON to the field at
// fault, and starts at the file's name in [Load].
//
// # Concurrency
//
// Every function is safe for concurrent use. Two runs that save one entry
// at once write one file, and the second reports that it wrote none.
//
// # Allocation contract
//
// [Verdict.Valid], [Verdict.String] and [Identity.Valid] allocate
// nothing. Every other function allocates the JSON it reads or writes.
//
// # Dependency position
//
// Imports the fault, choice, random and token packages of this module and
// the standard library.
package store
