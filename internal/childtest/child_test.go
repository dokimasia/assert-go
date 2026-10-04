// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package childtest_test

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/record"
)

// probe is a variable that a parent adds to the environment of its child.
const probe = "DOKIMI_ASSERT_CHILD_PROBE"

func TestChild(t *testing.T) {
	t.Parallel()

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		t.Run("runs the test alone in a child process with the variables that env adds", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				assert.Equal(t, os.Getenv(probe), "1", "the variable that env adds")
				return
			}
			out, err := childtest.Run(t, t.Name(), probe+"=1")
			assert.NoError(t, err, "the child passes:\n"+out)
			assert.Contains(t, out, "--- PASS: "+t.Name()+" ", "the child runs the test")
			assert.NotContains(t, out, "=== RUN   TestChildSwitch", "the child runs no other test")
		})
		t.Run("returns the error of a child whose test fails", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				t.Fail()
				return
			}
			out, err := childtest.Run(t, t.Name())
			var exit *exec.ExitError
			assert.True(t, errors.As(err, &exit), "the child exits with a status other than 0")
			assert.Contains(t, out, "--- FAIL: "+t.Name()+" ", "the child's test fails")
		})
	})

	t.Run("InChild", func(t *testing.T) {
		t.Parallel()

		t.Run("returns false in a process that no parent started for the test", func(t *testing.T) {
			t.Parallel()
			assert.False(t, childtest.InChild(t), "the parent process")
		})
	})
}

// TestChildSwitch checks that a child's switch is the one that env states,
// whatever the parent's switch is. It sets DOKIMI_ASSERT_RECORD for the whole
// test process, so it does not run in parallel. It checks with the package
// testing alone, because an assertion would read the switch that it sets.
func TestChildSwitch(t *testing.T) {
	if childtest.InChild(t) {
		if value, set := os.LookupEnv(record.Variable); set {
			t.Fatalf("the child's switch is %q, want it unset", value)
		}
		return
	}
	t.Setenv(record.Variable, "1")
	out, err := childtest.Run(t, t.Name())
	if err != nil || !strings.Contains(out, "--- PASS: "+t.Name()+" ") {
		t.Fatalf("the child exits with %v, want a pass:\n%s", err, out)
	}
}
