// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package childtest

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"regexp"
	"runtime/coverage"
	"strings"
	"sync"
	"testing"
	"time"

	"go.dokimi.dev/assert/internal/record"
)

// Variable is the environment variable whose value is the name of the test
// that a child process runs. [Run] sets it, and [InChild] reads it.
const Variable = "DOKIMI_ASSERT_CHILD"

// The bounds of a child process.
const (
	// limit is the longest that a child process runs.
	limit = time.Minute
	// grace is how long before the deadline of its parent a child ends.
	grace = 5 * time.Second
)

// metaWritten writes the coverage meta-data of the test binary once in the
// test process, into the directory of its coverage, before the first child
// runs. A child that finds the meta-data there writes none of its own. Two
// children that write it in one nanosecond write one temporary file, because
// the runtime names that file by the time alone, and the second fails to
// rename it.
var metaWritten sync.Once

// InChild reports whether t runs in the child process that [Run] started for
// it.
func InChild(t *testing.T) bool {
	t.Helper()
	return os.Getenv(Variable) == t.Name()
}

// Run runs the test name of the running test binary in a child process, and
// returns the child's output and the error of its exit, which is nil for an
// exit status of 0. name is a test's full name, as [testing.T.Name] returns
// it. The child runs that test alone, under -test.v=test2json, so its output
// states the test's status and its attributes in the form that test2json
// reads.
//
// The child's environment is the parent's without DOKIMI_ASSERT_RECORD, with
// [Variable] set to name and env added, so the child's switch is the one
// that env states. The child writes its coverage where the parent writes its
// own, after the parent has written the binary's coverage meta-data there.
// Where that write fails, each child writes the meta-data itself. A child
// ends within a minute, and 5 seconds before t's deadline at the latest.
func Run(t *testing.T, name string, env ...string) (string, error) {
	t.Helper()
	parts := strings.Split(name, "/")
	for i, part := range parts {
		parts[i] = "^" + regexp.QuoteMeta(part) + "$"
	}
	timeout := limit
	if deadline, ok := t.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline)-grace)
	}
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()

	argv := []string{
		"-test.run=" + strings.Join(parts, "/"), "-test.count=1", "-test.v=test2json",
		"-test.timeout=" + timeout.String(),
	}
	if dir := flag.Lookup("test.gocoverdir"); dir != nil && dir.Value.String() != "" {
		metaWritten.Do(func() { _ = coverage.WriteMetaDir(dir.Value.String()) })
		argv = append(argv, "-test.gocoverdir="+dir.Value.String())
	}
	cmd := exec.CommandContext(ctx, os.Args[0], argv...)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, record.Variable+"=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	cmd.Env = append(cmd.Env, Variable+"="+name)
	cmd.Env = append(cmd.Env, env...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
