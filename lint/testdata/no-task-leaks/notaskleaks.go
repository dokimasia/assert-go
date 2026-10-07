// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package notaskleaks

import (
	"runtime"
	"testing"

	"go.dokimi.dev/assert"
)

func serve() {}

func correct(t *testing.T) {
	check := assert.NoGoroutineLeaks(t, "Serve stops every goroutine that it starts")
	serve()
	check()
}

func counted(t *testing.T) {
	before := runtime.NumGoroutine()
	serve()
	assert.Equal(t, runtime.NumGoroutine(), before, "Serve stops every goroutine that it starts") // want `goroutine-leaks: state the check with NoGoroutineLeaks of runtime\.NumGoroutine\(\)`
	if runtime.NumGoroutine() > before {                                                          // want `goroutine-leaks: state the check with NoGoroutineLeaks`
		t.Fatal("Serve leaks a goroutine")
	}
}
