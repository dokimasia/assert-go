// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
)

// concurrentlyAllocs is the ceiling of the allocations of Concurrently for
// one client whose body returns at once.
const concurrentlyAllocs = 10

// The times of the driver's tests.
const (
	// patient is a time that no body of the tests needs.
	patient = time.Minute
	// brief is the time of a test whose body runs past it, and that a body
	// that returns at once meets on a loaded machine.
	brief = 100 * time.Millisecond
)

// TestConcurrently checks the driver that runs the clients of a history.
func TestConcurrently(t *testing.T) {
	t.Parallel()

	t.Run("Concurrently", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the outcome of each client in client order", func(t *testing.T) {
			t.Parallel()
			got := history.Concurrently(3, patient, func(client int) (any, error) {
				if client == 1 {
					return nil, errRefused
				}
				return client * 10, nil
			})
			assert.Equal(t, got, []history.Outcome{
				{Client: 0, Finished: true, Output: 0},
				{Client: 1, Finished: true, Error: errRefused},
				{Client: 2, Finished: true, Output: 20},
			}, "each body finished with its output or its error")
		})

		t.Run("runs the bodies at once", func(t *testing.T) {
			t.Parallel()
			var started sync.WaitGroup
			started.Add(4)
			got := history.Concurrently(4, patient, func(int) (any, error) {
				started.Done()
				started.Wait()
				return nil, nil
			})
			for _, o := range got {
				assert.True(t, o.Finished, "every body met the others")
			}
		})

		t.Run("returns a body that runs past the time as not finished", func(t *testing.T) {
			t.Parallel()
			release := make(chan struct{})
			got := history.Concurrently(2, brief, func(client int) (any, error) {
				if client == 0 {
					<-release
				}
				return client, nil
			})
			close(release)
			assert.Equal(t, got, []history.Outcome{
				{Client: 0},
				{Client: 1, Finished: true, Output: 1},
			}, "client 0 still waits when the time has passed")
		})

		t.Run("waits no time for a time of 0", func(t *testing.T) {
			t.Parallel()
			release := make(chan struct{})
			got := history.Concurrently(1, 0, func(int) (any, error) {
				<-release
				return nil, nil
			})
			close(release)
			assert.False(t, got[0].Finished, "the body waits on the release of the test")
		})

		t.Run("returns a body that ends its goroutine as finished without an output", func(t *testing.T) {
			t.Parallel()
			got := history.Concurrently(1, patient, func(int) (any, error) {
				runtime.Goexit()
				return 1, nil
			})
			assert.Equal(t, got, []history.Outcome{{Client: 0, Finished: true}}, "no output and no error")
		})

		t.Run("panics with the value of the lowest client that panicked once every body ended", func(t *testing.T) {
			t.Parallel()
			panicking := make(chan struct{})
			var waited atomic.Bool
			got := assert.Panics(t, func() {
				history.Concurrently(4, patient, func(client int) (any, error) {
					switch client {
					case 0:
						<-panicking
						waited.Store(true)
					case 2:
						panic(client)
					case 3:
						defer close(panicking)
						panic(client)
					}
					return nil, nil
				})
			}, "clients 2 and 3 panic")
			assert.Equal(t, got, any(2), "the value of client 2")
			assert.True(t, waited.Load(), "client 0, which ends after client 3 panics, ended first")
		})

		t.Run("drops the panic of a body that panics after the time passed", func(t *testing.T) {
			t.Parallel()
			release := make(chan struct{})
			recovered := make(chan struct{})
			got := history.Concurrently(1, brief, func(int) (any, error) {
				<-release
				defer close(recovered)
				panic("late")
			})
			close(release)
			<-recovered
			assert.False(t, got[0].Finished, "the body ran past the time")
		})

		tests := []struct {
			name    string
			clients int
			within  time.Duration
			want    string
		}{
			{
				name:    "panics for no client",
				clients: 0,
				within:  patient,
				want:    "history: Concurrently(0, 1m0s) starts no client",
			},
			{
				name:    "panics for a negative time",
				clients: 2,
				within:  -time.Second,
				want:    "history: Concurrently(2, -1s) waits a negative time",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got := assert.Panics(t, func() {
					history.Concurrently(tt.clients, tt.within, func(int) (any, error) { return nil, nil })
				}, "the driver refuses the arguments")
				assert.Equal(t, got, any(tt.want), "the panic names the call")
			})
		}
	})
}

// concurrentlyAllocsCases are the cases of the allocation ceiling of
// Concurrently.
var concurrentlyAllocsCases = []alloctest.Case{
	{
		Name: "Concurrently",
		Call: func(assert.TB) {
			history.Concurrently(1, patient, func(int) (any, error) { return nil, nil })
		},
		Allocs: concurrentlyAllocs,
	},
}

// TestConcurrentlyAllocs checks the allocation ceiling of Concurrently.
func TestConcurrentlyAllocs(t *testing.T) {
	alloctest.Check(t, concurrentlyAllocsCases)
}

// BenchmarkConcurrently measures Concurrently under its ceiling.
func BenchmarkConcurrently(b *testing.B) {
	for _, c := range concurrentlyAllocsCases {
		b.Run(c.Name, func(b *testing.B) { alloctest.Measure(b, c) })
	}
}
