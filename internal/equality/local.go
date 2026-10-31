// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package equality

import (
	"reflect"
	"time"
)

// local is the location of time.Local when the process starts. The time
// package fills its fields on the first call of one of its methods, such as
// the first formatting of a local time.
var local = time.Local

// localAddress is the address of local.
var localAddress = reflect.ValueOf(local).Pointer()

// target returns the target of p, a pointer that is not nil. When the
// target is local, target calls a method of local first, so the walk reads
// its fields after the time package fills them.
func target(p reflect.Value) reflect.Value {
	//dokimi:mutate-skip ror-false,ror-true,sbr-delete: only the race detector observes the order of the reads, and a call for another pointer changes nothing
	if p.Pointer() == localAddress {
		_ = local.String()
	}
	return p.Elem()
}
