// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package zone

import (
	"time"

	"go.dokimi.dev/assert/internal/fault"
)

// Locations returns the location of each zone of the list by name, as load
// resolves it. A caller passes time.LoadLocation to resolve the zones in
// the platform's time-zone database.
//
// # Errors
//
// It returns a fault at the name of the first zone that load cannot
// resolve, whose cause is the error of load.
//
// # Allocation contract
//
// Locations allocates the map that it returns, and what load allocates:
// four allocations for a load that allocates nothing.
func Locations(load func(name string) (*time.Location, error)) (map[string]*time.Location, error) {
	listed := List()
	out := make(map[string]*time.Location, len(listed))
	for _, z := range listed {
		loc, err := load(z.Name)
		if err != nil {
			return nil, fault.At(fault.New("the time-zone database has no such zone").Because(err), fault.Key(z.Name))
		}
		out[z.Name] = loc
	}
	return out, nil
}
