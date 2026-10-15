// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package realtime reads the time that passes in the process, which a
// testing/synctest bubble does not fake. The limits that bound how long a
// run of this module takes read it: the budget of a campaign, the time of a
// shrink and the time limit of a history check.
//
// # Dependency position
//
// Imports the standard library's syscall and time.
package realtime
