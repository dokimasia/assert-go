// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import "time"

// Variant is a value of an enum shape, as [OfShape] decodes one: the
// variant's name, and its payload when the variant has one. HasPayload
// keeps a variant without a payload apart from one whose payload is an
// absent optional, whose Payload is nil.
type Variant struct {
	// Name is the variant's name.
	Name string
	// Payload is the variant's payload, as OfShape decodes its shape, when
	// HasPayload is set.
	Payload any
	// HasPayload reports whether the variant has a payload.
	HasPayload bool
}

// WallTime is a value of the wall-time shape: a wall time, and the zone
// that the code under test resolves it in. A wall time inside a gap of the
// zone has no instant, and one inside a fold has two, so the value states
// no instant.
//
// [Of] reads a WallTime as the wall-time shape at nanoseconds, and the tag
// unit states a coarser unit, as for a time.Time.
type WallTime struct {
	// Local is the wall time as a time.Time in UTC, whose fields state the
	// date and the time of day on a clock in Zone.
	Local time.Time
	// Zone is the zone of the wall time.
	Zone *time.Location
}
