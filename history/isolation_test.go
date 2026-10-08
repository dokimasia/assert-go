// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"math/rand/v2"
	"runtime"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/history"
	"go.dokimi.dev/assert/internal/alloctest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/matcher"
	"go.dokimi.dev/assert/internal/matchertest"
)

// The allocation ceilings of a passing check.
const (
	// serializableAllocs is the ceiling of the allocations of Serializable
	// over serialHistory.
	serializableAllocs = 120
	// snapshotIsolationAllocs is the ceiling of the allocations of
	// HasSnapshotIsolation over skewHistory.
	snapshotIsolationAllocs = 110
)

// writeSkewJSON is the detail of the record of Serializable over writeSkew,
// in the history's JSON form, as the reference computes it.
const writeSkewJSON = `{"anomaly":"G2","kinds":["G2"],"transactions":[` +
	`{"call":0,"completion":1,"kind":"ok","process":0,"args":[` +
	`{"type":"list","items":[{"type":"string","value":"read"},{"type":"string","value":"x"},{"type":"null"}]},` +
	`{"type":"list","items":[{"type":"string","value":"append"},{"type":"string","value":"y"},` +
	`{"type":"int","value":1}]}],"output":{"type":"list","items":[` +
	`{"type":"list","items":[{"type":"string","value":"read"},{"type":"string","value":"x"},` +
	`{"type":"list","items":[]}]},` +
	`{"type":"list","items":[{"type":"string","value":"append"},{"type":"string","value":"y"},` +
	`{"type":"int","value":1}]}]}},` +
	`{"call":2,"completion":3,"kind":"ok","process":1,"args":[` +
	`{"type":"list","items":[{"type":"string","value":"read"},{"type":"string","value":"y"},{"type":"null"}]},` +
	`{"type":"list","items":[{"type":"string","value":"append"},{"type":"string","value":"x"},` +
	`{"type":"int","value":2}]}],"output":{"type":"list","items":[` +
	`{"type":"list","items":[{"type":"string","value":"read"},{"type":"string","value":"y"},` +
	`{"type":"list","items":[]}]},` +
	`{"type":"list","items":[{"type":"string","value":"append"},{"type":"string","value":"x"},` +
	`{"type":"int","value":2}]}]}}],` +
	`"cycle":[{"call":0,"relations":["rw"]},{"call":2,"relations":["rw"]}],"explanation":[` +
	`{"from":0,"to":2,"relation":"rw","key":{"type":"string","value":"x"},"value":null,` +
	`"next":{"type":"int","value":2}},` +
	`{"from":2,"to":0,"relation":"rw","key":{"type":"string","value":"y"},"value":null,` +
	`"next":{"type":"int","value":1}}]}`

// TestIsolation checks the two isolation checks: their verdicts, their
// records and their faults.
func TestIsolation(t *testing.T) {
	t.Parallel()

	t.Run("Serializable", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a serial history with a call record of a pass", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			history.Serializable(rec, serialHistory, isolationContract)
			assert.False(t, rec.Failed(), "each transaction read the appends before it")
			got := recordOf(t, rec)
			assert.Equal(t, []string{got.Assertion, got.Verdict}, []string{"serializable", "pass"},
				"the call record of the pass")
		})

		t.Run("fails a write skew with one record of serializable at the call", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			_, file, line, _ := runtime.Caller(0)
			history.Serializable(rec, skewHistory, isolationContract)
			failures := rec.Failures()
			assert.Length(t, failures, 1, "one record")
			got := failures[0]
			assert.Equal(t, []any{got.Assertion, got.Contract, got.Where},
				[]any{"serializable", isolationContract, assert.Where{File: file, Line: line + 1}},
				"the assertion, the contract and the line of the call")
		})

		t.Run("states the detail in the history's JSON form in the call record", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			history.Serializable(rec, skewHistory, isolationContract)
			assert.Equal(t, string(recordOf(t, rec).Detail), writeSkewJSON, "the detail of the reference")
		})

		t.Run("sends the sentence of the record through Fatalf to a seat without Report", func(t *testing.T) {
			t.Parallel()
			seat := &fatalSeat{}
			history.Serializable(seat, skewHistory, isolationContract)
			rec := assert.NewRecorder()
			history.Serializable(rec, skewHistory, isolationContract)
			assert.Equal(t, seat.texts, []string{matcher.Render(rec.Failures()[0])}, "the sentence of the record")
		})

		t.Run(
			"numbers the transactions of a history that FromIntervals built by their invocations",
			func(t *testing.T) {
				t.Parallel()
				first, firstOutput := statedBy([][]any{readOf("x"), appendOf("y", 1)})
				second, secondOutput := statedBy([][]any{readOf("y"), appendOf("x", 2)})
				h, err := history.FromIntervals([]history.Interval{
					{
						Client: 0, Operation: "txn", Args: first, Keys: []any{"x", "y"}, Start: 0, End: 3,
						Kind: history.OK, Output: firstOutput,
					},
					{
						Client: 1, Operation: "txn", Args: second, Keys: []any{"y", "x"}, Start: 1, End: 2,
						Kind: history.OK, Output: secondOutput,
					},
				})
				assert.NoError(t, err, "each client makes one call")
				assert.Equal(t, isolationOf(history.Serializable, h)[cycleField], any([]history.Link{
					{Call: 0, Relations: []history.Relation{history.RW}},
					{Call: 1, Relations: []history.Relation{history.RW}},
				}), "the second invocation is the event after the first")
			},
		)

		t.Run("returns a fault for a nil history", func(t *testing.T) {
			t.Parallel()
			expectFault(t, isolationFault(t, history.Serializable, nil),
				fault.Error{Op: serializableOp, Reason: "the history is nil"})
		})
	})

	t.Run("HasSnapshotIsolation", func(t *testing.T) {
		t.Parallel()

		t.Run("passes a write skew with a call record of a pass", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			history.HasSnapshotIsolation(rec, skewHistory, isolationContract)
			assert.False(t, rec.Failed(), "the two rw edges of the cycle are adjacent")
			got := recordOf(t, rec)
			assert.Equal(t, []string{got.Assertion, got.Verdict}, []string{"snapshot-isolation", "pass"},
				"the call record of the pass")
		})

		t.Run("fails a long fork with one record of snapshot-isolation", func(t *testing.T) {
			t.Parallel()
			rec := assert.NewRecorder()
			history.HasSnapshotIsolation(rec, longFork(), isolationContract)
			failures := rec.Failures()
			assert.Length(t, failures, 1, "one record")
			assert.Equal(t, []any{failures[0].Assertion, failures[0].Detail[anomalyField]},
				[]any{"snapshot-isolation", history.GNonadjacent}, "the assertion and the anomaly")
		})

		t.Run("returns a fault for a nil history", func(t *testing.T) {
			t.Parallel()
			expectFault(t, isolationFault(t, history.HasSnapshotIsolation, nil),
				fault.Error{Op: snapshotIsolationOp, Reason: "the history is nil"})
		})
	})
}

// serialHistory is a history in which each transaction reads the appends of
// the transactions before it.
var serialHistory = func() *history.History {
	h := history.New()
	transact(h, 0, history.OK, appendOf("x", 1))
	transact(h, 1, history.OK, readOf("x", 1), appendOf("x", 2))
	transact(h, 0, history.OK, readOf("x", 1, 2))
	return h
}()

// skewHistory is the history of a write skew.
var skewHistory = writeSkew()

// isolationAllocs are the cases of the allocation ceilings of a passing
// check of each level.
var isolationAllocs = []alloctest.Case{
	{
		Name:   "Serializable",
		Call:   func(tb assert.TB) { history.Serializable(tb, serialHistory, isolationContract) },
		Allocs: serializableAllocs,
	},
	{
		Name:   "HasSnapshotIsolation",
		Call:   func(tb assert.TB) { history.HasSnapshotIsolation(tb, skewHistory, isolationContract) },
		Allocs: snapshotIsolationAllocs,
	},
}

// TestIsolationAllocs checks the allocation ceiling of a passing check of
// each level.
func TestIsolationAllocs(t *testing.T) {
	alloctest.Check(t, isolationAllocs)
}

// BenchmarkIsolation measures a passing check of each level under its
// ceiling, and the time of the checks of three histories of 10,000
// transactions or more: Serializable over the history of a serializable
// store and over the history of two chains, and HasSnapshotIsolation over
// the history of a snapshot store. The definition's executable reference
// checks the same three histories in 0.49 s, 2.2 s and 0.44 s on one
// processor. The allocations of a check that fails count the setup of the
// module's first failure in a run of one iteration, and those of a large
// history depend on its size, so those three state no ceiling.
func BenchmarkIsolation(b *testing.B) {
	b.Run("Serializable", func(b *testing.B) {
		b.Run("pass", func(b *testing.B) { alloctest.Measure(b, isolationAllocs[0]) })
		b.Run("store", func(b *testing.B) { checkLarge(b, history.Serializable, storeHistory(false)) })
		b.Run("chains", func(b *testing.B) {
			checkLarge(b, history.Serializable, chainHistory(), history.GNonadjacent, history.G2)
		})
	})
	b.Run("HasSnapshotIsolation", func(b *testing.B) {
		b.Run("pass", func(b *testing.B) { alloctest.Measure(b, isolationAllocs[1]) })
		b.Run("store", func(b *testing.B) { checkLarge(b, history.HasSnapshotIsolation, storeHistory(true)) })
	})
}

// checkLarge measures the check of h with level on a seat that writes no
// call record, and checks the kinds of the record of the last check: want,
// or no record for no kind.
func checkLarge(b *testing.B, level func(assert.TB, *history.History, string), h *history.History,
	want ...history.Anomaly,
) {
	b.Helper()
	seat := &matchertest.Seat{}
	c := bench.Start(b)
	defer c.End()
	for c.Loop() {
		level(seat, h, isolationContract)
	}
	records := seat.Records()
	if len(want) == 0 {
		assert.Empty(b, records, "the check passes")
		return
	}
	assert.Equal(b, records[len(records)-1].Detail[kindsField], any(want), "the kinds of the record")
}

// The parameters of the histories of the simulated stores.
const (
	// storeTransactions is the number of transactions of a history.
	storeTransactions = 10_000
	// storeClients is the number of clients that run them.
	storeClients = 8
	// storeKeys is the number of active keys.
	storeKeys = 16
	// storeRetire is the number of appends after which a key retires, and a
	// fresh key takes its place.
	storeRetire = 32
	// storeOperations is the most micro-operations of one transaction.
	storeOperations = 4
)

// store is a simulated store of lists of ints, and the history that its
// clients record.
type store struct {
	// snapshot reports a store that reads from the state at a transaction's
	// start, and false a store that runs a transaction whole at its commit.
	snapshot bool
	// rng is the random source of the steps of the run.
	rng *rand.Rand
	// h is the history.
	h *history.History
	// active are the active keys, and fresh the next key that takes the place
	// of a retired one.
	active []int
	fresh  int
	// state is the list of each key, of the transactions that committed.
	state map[int][]int
	// commits is the number of commits, and lastCommit the number of the
	// commit that last appended to each key.
	commits    int
	lastCommit map[int]int
	// appends is the number of appends that the transactions made to each
	// key, which numbers the values.
	appends map[int]int
	// running maps each client with a transaction in progress to it, and
	// started is the number of transactions that started.
	running map[int]*running
	started int
}

// running is a transaction in progress.
type running struct {
	// call is the transaction's call.
	call history.Call
	// operations are its micro-operations, and done those that ran, each
	// read with its list.
	operations []any
	done       []any
	// view is the list of each key that it touches, at its start, and
	// commits the number of commits before its start.
	view    map[int][]int
	commits int
	// appended are the values that it appended to each key.
	appended map[int][]int
}

// storeHistory returns the history of storeTransactions transactions that a
// simulated store records, from a fixed seed. Each step picks one of
// storeClients clients at random. An idle client starts a transaction of one
// to storeOperations micro-operations on the active keys, each a read or an
// append with equal chance, and a busy client advances its transaction.
//
// A serializable store runs a transaction whole at its first advance and
// commits it. A snapshot store runs one micro-operation per advance, and
// reads the state at the transaction's start with the transaction's own
// appends. It aborts a transaction at its commit when another transaction
// committed an append to one of its keys after its start.
func storeHistory(snapshot bool) *history.History {
	s := &store{
		snapshot: snapshot, rng: rand.New(rand.NewPCG(storeTransactions, 0)), h: history.New(),
		active: make([]int, storeKeys), fresh: storeKeys, state: map[int][]int{}, lastCommit: map[int]int{},
		appends: map[int]int{}, running: map[int]*running{},
	}
	for key := range s.active {
		s.active[key] = key
	}
	for s.started < storeTransactions || len(s.running) > 0 {
		client := s.rng.IntN(storeClients)
		if t, busy := s.running[client]; busy {
			s.advance(client, t)
		} else if s.started < storeTransactions {
			s.start(client)
		}
	}
	return s.h
}

// key returns an active key at random. A key that had storeRetire appends
// retires, and a fresh key takes its place.
func (s *store) key() int {
	slot := s.rng.IntN(len(s.active))
	if s.appends[s.active[slot]] >= storeRetire {
		s.active[slot] = s.fresh
		s.fresh++
	}
	return s.active[slot]
}

// start starts a transaction of client.
func (s *store) start(client int) {
	operations := make([]any, 1+s.rng.IntN(storeOperations))
	var keys []any
	for i := range operations {
		key := s.key()
		operations[i] = []any{"read", key, nil}
		if s.rng.IntN(2) == 0 {
			s.appends[key]++
			operations[i] = appendOf(key, s.appends[key])
		}
		if !slices.Contains(keys, any(key)) {
			keys = append(keys, key)
		}
	}
	t := &running{
		call: s.h.Invoke(client, "txn", operations, keys...), operations: operations, view: map[int][]int{},
		commits: s.commits, appended: map[int][]int{},
	}
	for _, key := range keys {
		t.view[key.(int)] = s.state[key.(int)]
	}
	s.running[client] = t
	s.started++
}

// advance runs the next micro-operation of t, the transaction of client, or
// commits it when every micro-operation ran. A serializable store runs every
// micro-operation at once, and commits.
func (s *store) advance(client int, t *running) {
	if !s.snapshot {
		for _, operation := range t.operations {
			t.done = append(t.done, t.run(operation.([]any), s.state))
		}
		s.commit(client, t)
		return
	}
	if len(t.done) < len(t.operations) {
		t.done = append(t.done, t.run(t.operations[len(t.done)].([]any), t.view))
		return
	}
	s.commit(client, t)
}

// run runs one micro-operation of t on the lists of view, and returns the
// micro-operation of t's output. A read returns the list of view with t's
// own appends after it.
func (t *running) run(operation []any, view map[int][]int) []any {
	key := operation[1].(int)
	if operation[0] == "append" {
		t.appended[key] = append(t.appended[key], operation[2].(int))
		return operation
	}
	return []any{"read", key, slices.Concat(view[key], t.appended[key])}
}

// commit completes t, the transaction of client. A snapshot store aborts t
// when another transaction committed an append to one of t's keys after t
// started. Otherwise t's appends join the state.
func (s *store) commit(client int, t *running) {
	delete(s.running, client)
	for key := range t.appended {
		if s.snapshot && s.lastCommit[key] > t.commits {
			t.call.Fail(errAborted)
			return
		}
	}
	s.commits++
	for key, values := range t.appended {
		s.state[key] = append(s.state[key], values...)
		s.lastCommit[key] = s.commits
	}
	t.call.OK(t.done)
}

// The parameters of the history of two chains.
const (
	// chainLength is the number of transactions of each chain.
	chainLength = 5_000
	// chainReads is the number of keys that each transaction of the first
	// chain reads.
	chainReads = 2
)

// chainHistory returns a history of 2·chainLength + 1 transactions, which
// commit one after another, from a fixed seed. The transactions of chain A
// append 1 to chainLength to the key "a", and those of chain B append them
// to "b". The j-th transaction of B also appends 1 to the key j. Each
// transaction of A reads chainReads of the keys 1 to chainLength at random,
// and finds them empty. The first transaction of A appends 1 to "c", which
// the last transaction of B reads empty. A last transaction reads "a" and
// "b" whole.
//
// Every cycle then has two read-write edges or more, so the search for a
// G-single cycle tries each read-write edge and walks chain B from it.
func chainHistory() *history.History {
	rng := rand.New(rand.NewPCG(chainLength, 0))
	h := history.New()
	whole := make([]any, chainLength)
	for i := range whole {
		whole[i] = i + 1
	}
	for i := 1; i <= chainLength; i++ {
		operations := [][]any{appendOf("a", i)}
		if i == 1 {
			operations = append(operations, appendOf("c", 1))
		}
		for range chainReads {
			operations = append(operations, readOf(1+rng.IntN(chainLength)))
		}
		transact(h, 0, history.OK, operations...)
	}
	for j := 1; j <= chainLength; j++ {
		operations := [][]any{appendOf("b", j), appendOf(j, 1)}
		if j == chainLength {
			operations = append(operations, readOf("c"))
		}
		transact(h, 0, history.OK, operations...)
	}
	transact(h, 0, history.OK, readOf("a", whole...), readOf("b", whole...))
	return h
}
