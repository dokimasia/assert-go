// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package zone

import (
	_ "embed"
	"encoding/json"
	"sync"
)

// table is the definition's zone table.
//
//go:embed zones.json
var table []byte

// Change is one offset change of a zone: its first second, and the offsets
// before and after it.
type Change struct {
	// At is the change's first second, in seconds since
	// 1970-01-01T00:00:00Z.
	At int64
	// Before is the offset before the change, in seconds east of UTC.
	Before int64
	// After is the offset from the change on, in seconds east of UTC.
	After int64
}

// Zone is a zone of the list, and its offset changes in time order.
type Zone struct {
	// Name is the zone's name in the IANA time-zone database.
	Name string
	// Changes are the zone's offset changes, in time order.
	Changes []Change
}

// list parses the embedded table once. The table is the definition's,
// whose digest a test compares with the vendored manifest's, so it parses,
// and each of its changes states an instant and two offsets.
var list = sync.OnceValue(func() []Zone {
	var doc struct {
		Zones []struct {
			Name    string     `json:"name"`
			Changes [][3]int64 `json:"changes"`
		} `json:"zones"`
	}
	_ = json.Unmarshal(table, &doc)
	zones := make([]Zone, len(doc.Zones))
	for i, z := range doc.Zones {
		changes := make([]Change, len(z.Changes))
		for j, c := range z.Changes {
			changes[j] = Change{At: c[0], Before: c[1], After: c[2]}
		}
		zones[i] = Zone{Name: z.Name, Changes: changes}
	}
	return zones
})

// List returns the zone list, in the definition's order. Every call returns
// the same slice, which a caller must not change.
func List() []Zone {
	return list()
}
