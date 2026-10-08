// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package literal

// Field is one field of a [Record]: its name and its value.
type Field struct {
	// Name is the field's name.
	Name string
	// Value is the field's value.
	Value any
}

// Record is a value of the record shape: its fields in declaration order.
// No two fields share a name.
type Record struct {
	// Fields are the record's fields, in declaration order.
	Fields []Field
}

// Entry is one entry of [Pairs]: its key and its value.
type Entry struct {
	// Key is the entry's key.
	Key any
	// Value is the entry's value.
	Value any
}

// Pairs is a value of the map shape: its entries in the order that a case
// generated them. No two entries have equal keys.
type Pairs struct {
	// Entries are the map's entries, in the order of generation.
	Entries []Entry
}

// Variant is a value of the enum shape: the variant's name, and its payload
// when the variant has one. HasPayload keeps a variant without a payload
// apart from one whose optional payload is absent, which has a nil Payload.
type Variant struct {
	// Name is the variant's name.
	Name string
	// Payload is the variant's payload, when HasPayload is set.
	Payload any
	// HasPayload reports whether the variant has a payload.
	HasPayload bool
}
