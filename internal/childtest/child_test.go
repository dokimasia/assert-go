// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package childtest_test

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

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
		t.Run("gives a child half of the time that a deadline under 10 seconds leaves", func(t *testing.T) {
			t.Parallel()
			switch {
			case !childtest.InChild(t):
				out, err := childtest.RunFlags(t, t.Name(), []string{"-test.timeout=4s"}, probe+"=parent")
				assert.NoError(t, err, "the test with a deadline 4 seconds away passes:\n"+out)
			case os.Getenv(probe) == "parent":
				out, err := childtest.Run(t, t.Name(), probe+"=child")
				assert.NoError(t, err, "its child passes:\n"+out)
			default:
				deadline, _ := t.Deadline()
				assert.InRange(t, time.Until(deadline), 0, float64(2*time.Second), "the child has half of 4 seconds")
			}
		})
	})

	t.Run("RunFlags", func(t *testing.T) {
		t.Parallel()

		t.Run("passes the flags after its own, so -test.count=2 runs the test twice", func(t *testing.T) {
			t.Parallel()
			if childtest.InChild(t) {
				return
			}
			out, err := childtest.RunFlags(t, t.Name(), []string{"-test.count=2"})
			assert.NoError(t, err, "the child passes:\n"+out)
			assert.Equal(t, strings.Count(out, "--- PASS: "+t.Name()+" "), 2, "the child runs the test twice")
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
