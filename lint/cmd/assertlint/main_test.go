// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fixture is the package of the test module that the command runs over.
const fixture = "../../testdata/command"

// diagnostics are the lines that the command writes for the fixture, each
// after the path of the fixture's file.
var diagnostics = []string{
	":13:2: compare: state the check with Equal\n",
	":14:2: order: state the check with InRange\n",
}

func TestAssertlint(t *testing.T) {
	t.Parallel()
	// Two runs of a build with coverage that write their meta-data in one
	// nanosecond write one temporary file, and the second fails to rename it.
	// One run of the build with coverage therefore covers main. Every other
	// run uses an ordinary build: the run with -fix, and the vet and fix tool
	// that the go command runs once for each package, several at once.
	covered := build(t, coverDir() != "")
	command := covered
	if coverDir() != "" {
		command = build(t, false)
	}
	t.Run("main", func(t *testing.T) {
		t.Parallel()
		t.Run("reports each hand-written check and exits with status 3", func(t *testing.T) {
			t.Parallel()
			dir := workspace(t)
			stderr, code := run(t, dir, covered, "./...")
			if code != 3 || !diagnosed(stderr) {
				t.Errorf("assertlint exits with %d and writes\n%s\nwant 3 and %q", code, stderr, diagnostics)
			}
		})
		t.Run("applies each fix under -fix and exits with status 0", func(t *testing.T) {
			t.Parallel()
			dir := workspace(t)
			stderr, code := run(t, dir, command, "-fix", "./...")
			if code != 0 || stderr != "" {
				t.Errorf("assertlint -fix exits with %d and writes %q, want 0 and nothing", code, stderr)
			}
			fixed(t, dir)
		})
		t.Run("reports each hand-written check to go vet, which exits with status 1", func(t *testing.T) {
			t.Parallel()
			dir := workspace(t)
			stderr, code := run(t, dir, "go", "vet", "-vettool="+command, "./...")
			if code != 1 || !diagnosed(stderr) {
				t.Errorf("go vet exits with %d and writes\n%s\nwant 1 and %q", code, stderr, diagnostics)
			}
		})
		t.Run("applies each fix under go fix", func(t *testing.T) {
			t.Parallel()
			dir := workspace(t)
			stderr, code := run(t, dir, "go", "fix", "-fixtool="+command, "./...")
			if code != 0 || stderr != "" {
				t.Errorf("go fix exits with %d and writes %q, want 0 and nothing", code, stderr)
			}
			fixed(t, dir)
		})
	})
}

// build builds the command into a directory of the test, and returns its
// path. With cover, it builds the command with coverage in the test's mode,
// so the command's counters join the test's in -test.gocoverdir.
func build(t *testing.T, cover bool) string {
	t.Helper()
	command := filepath.Join(t.TempDir(), "assertlint")
	if runtime.GOOS == "windows" {
		command += ".exe"
	}
	args := []string{"build", "-o", command}
	if cover {
		args = append(args, "-cover", "-covermode="+testing.CoverMode())
	}
	out, err := exec.CommandContext(t.Context(), "go", append(args, ".")...).CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return command
}

// workspace copies the fixture into a module of the test's own, which
// requires go.dokimi.dev/assert from this repository, and returns its
// directory.
func workspace(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatalf("find the repository's root: %v", err)
	}
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(fixture)); err != nil {
		t.Fatalf("copy the fixture: %v", err)
	}
	module := "module example.test/command\n\ngo 1.27.2\n\n" +
		"require go.dokimi.dev/assert v0.0.0-00010101000000-000000000000\n\n" +
		"replace go.dokimi.dev/assert => " + filepath.ToSlash(root) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o644); err != nil {
		t.Fatalf("write the module: %v", err)
	}
	return dir
}

// run runs the program with args in dir, and returns what it writes to its
// standard error and its exit status. The program loads the module of dir
// alone, from the files on disk.
func run(t *testing.T, dir, program string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), program, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOFLAGS=")
	if coverage := coverDir(); coverage != "" {
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+coverage)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	var exit *exec.ExitError
	if err := cmd.Run(); err != nil && !errors.As(err, &exit) {
		t.Fatalf("run %s: %v", program, err)
	}
	return stderr.String(), cmd.ProcessState.ExitCode()
}

// diagnosed reports whether stderr is a line of each diagnostic, in order,
// each after a path whose last element is the fixture's file. The command
// writes the path in full and go vet relative to its directory, each with
// the separator of the platform.
func diagnosed(stderr string) bool {
	lines := strings.SplitAfter(stderr, "\n")
	if len(lines) != len(diagnostics)+1 || lines[len(diagnostics)] != "" {
		return false
	}
	for i, diagnostic := range diagnostics {
		file, ok := strings.CutSuffix(lines[i], diagnostic)
		if !ok || filepath.Base(file) != "command.go" {
			return false
		}
	}
	return true
}

// fixed fails the test where the fixture's file in dir differs from its
// golden file.
func fixed(t *testing.T, dir string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(dir, "command.go"))
	if err != nil {
		t.Fatalf("read the fixed file: %v", err)
	}
	want, err := os.ReadFile(filepath.Join(fixture, "command.go.golden"))
	if err != nil {
		t.Fatalf("read the golden file: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("the fix leaves\n%s\nwant\n%s", got, want)
	}
}

// coverDir returns the directory of the test's coverage counters, and empty
// for a run without coverage.
func coverDir() string {
	if dir := flag.Lookup("test.gocoverdir"); dir != nil {
		return dir.Value.String()
	}
	return ""
}
