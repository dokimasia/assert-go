// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package total

import (
	"testing"

	. "go.dokimi.dev/assert"
)

func dotted(t *testing.T, names []string) {
	for _, name := range names { // want `total: state the check with Total`
		NoError(t, valid(name), "every name is valid")
	}
}
