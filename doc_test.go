// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package assert_test

import (
	"fmt"

	"go.dokimi.dev/assert"
)

// A Recorder stands in for *testing.T so the example prints the
// outcome instead of failing.
func ExampleNewRecorder() {
	r := assert.NewRecorder()
	fmt.Println(r.Failed())
	// Output: false
}
