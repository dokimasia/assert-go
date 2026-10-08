// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package eventually

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.dokimi.dev/assert"
)

// pause is the length of each wait that a test measures.
const pause = 20 * time.Millisecond

func ready() bool { return true }

func count() int { return 3 }

func correct(t *testing.T) {
	assert.Eventually(t, time.Second, 10*time.Millisecond, func(tb assert.TB) {
		assert.True(tb, ready(), "the server is ready")
	}, "the server becomes ready within a second")
}

func polled(t *testing.T) {
	for i := 0; i < 100; i++ { // want `eventually: state the check with EventuallyTrue or Eventually`
		if ready() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	for range 3 {
		t.Log("waiting")
	}
}

// created waits up to ten seconds for the file at p.
func created(t *testing.T, p string) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) { // want `eventually: state the check with EventuallyTrue or Eventually`
		if _, err := os.Stat(p); err == nil {
			return
		}
	}
	t.Fatalf("%s was not created within ten seconds", p)
}

// counted waits until the count is at its limit.
func counted(t *testing.T, stale string) {
	limit := 3
	for count() < limit { // want `eventually: state the check with EventuallyTrue or Eventually`
		_ = os.Remove(stale)
		time.Sleep(time.Millisecond)
	}
	for attempt := 1; count() < limit; attempt++ { // want `eventually: state the check with EventuallyTrue or Eventually`
		t.Log(attempt)
		time.Sleep(time.Millisecond)
	}
	for i := 0; i < 10 && count() < limit; i++ { // want `eventually: state the check with EventuallyTrue or Eventually`
		time.Sleep(time.Millisecond)
	}
	waiting := true
	for waiting && count() < limit { // want `eventually: state the check with EventuallyTrue or Eventually`
		time.Sleep(time.Millisecond)
		waiting = !ready()
	}
}

// awaited waits for done, and breaks out of the loop through its label.
func awaited(t *testing.T, done <-chan struct{}, states []bool) {
wait:
	for { // want `eventually: state the check with EventuallyTrue or Eventually`
		select {
		case <-done:
			break wait
		default:
		}
		time.Sleep(time.Millisecond)
	}
outer:
	for { // want `eventually: state the check with EventuallyTrue or Eventually`
	inner:
		for _, s := range states {
			if s {
				break outer
			}
			if !ready() {
				continue inner
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Log("done")
}

// subtests waits in a subtest, and spaces the subtests of a loop.
func subtests(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		for { // want `eventually: state the check with EventuallyTrue or Eventually`
			if ready() {
				return
			}
			time.Sleep(time.Millisecond)
		}
	})
	for i := range 3 {
		t.Run("round", func(t *testing.T) {
			if !ready() {
				return
			}
			t.Log(i)
		})
		time.Sleep(time.Millisecond)
	}
}

// spaced sleeps for a count of rounds, which waits for no condition.
func spaced(t *testing.T, states []bool, ticks chan<- time.Time) {
	for range 2 {
		began := time.Now()
		time.Sleep(pause)
		t.Log(time.Since(began))
	}
	for i := 0; i < 2; i++ {
		time.Sleep(pause)
	}
	for left := 2; 0 < left; left-- {
		time.Sleep(pause)
	}
	i := 0
	for i < 2 {
		time.Sleep(pause)
		i++
	}
	for range 3 {
		for _, s := range states {
			if s {
				break
			}
		}
		switch {
		case ready():
			if len(states) == 0 {
				break
			}
			t.Log("ready")
		}
	scan:
		for _, s := range states {
			if s {
				break scan
			}
		}
		if !ready() {
			continue
		}
		time.Sleep(pause)
	}
	go func() {
		for {
			time.Sleep(time.Millisecond)
			ticks <- time.Now()
		}
	}()
}

// appending writes two files apart in each of five rounds for a child
// process of a lock test. No test is in scope.
func appending(dir string) error {
	for range 5 {
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o600); err != nil {
			return err
		}
		time.Sleep(pause)
		if err := os.WriteFile(filepath.Join(dir, "b.txt"), nil, 0o600); err != nil {
			return err
		}
	}
	return nil
}
