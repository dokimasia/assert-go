// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package fake

import "testing"

func True(tb testing.TB, cond bool, msg string) {
	tb.Helper()
	if !cond { // want `condition: state the check with True`
		tb.Fatal(msg)
	}
}
