// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/internal/prop/store"
)

// The store of a test, and the definition version its entries state.
const (
	// definition is the version of the definition that this package
	// implements, which every entry it writes states.
	definition = "1.4.0"
	// storeRoot is the directory of the stores of a package's tests,
	// relative to the package's directory, beside testdata/golden.
	storeRoot = "testdata/prop"
)

// named is a seat whose test has a name, as testing.TB has.
type named interface {
	// Name returns the test's name, each subtest's after its parent's and a
	// slash.
	Name() string
}

// cleaner is a seat with cleanup functions, which run when its test ends,
// as testing.TB has them.
type cleaner interface {
	// Cleanup registers f to run when the test ends.
	Cleanup(f func())
}

// claimed is a property's claim on the entries of its contract in one
// store.
type claimed struct {
	// dir is the store's directory.
	dir string
	// contract is the property's contract.
	contract string
}

// claims are the claims of the properties of the tests that run, which
// each test releases when it ends.
var claims = struct {
	// mu guards held.
	mu sync.Mutex
	// held are the claims held.
	held map[claimed]struct{}
}{held: make(map[claimed]struct{})}

// directoryOf returns the directory of the store of a run under c on tb:
// the one that Store states, then testdata/prop/<name> for a seat with a
// name, and otherwise the empty string, which states no store.
func directoryOf(tb assert.TB, c config) string {
	if c.storeSet {
		return c.store
	}
	if n, ok := tb.(named); ok {
		return filepath.Join(storeRoot, n.Name())
	}
	return ""
}

// claim claims the entries of contract in the store dir for the test of tb
// until the test ends, and reports false when another property of a test
// that runs holds them. A run without a store, and a seat without Cleanup,
// claim nothing and report true.
func claim(tb assert.TB, dir, contract string) bool {
	c, ok := tb.(cleaner)
	if !ok || dir == "" {
		return true
	}
	key := claimed{dir: dir, contract: contract}
	claims.mu.Lock()
	defer claims.mu.Unlock()
	if _, held := claims.held[key]; held {
		return false
	}
	claims.held[key] = struct{}{}
	c.Cleanup(func() {
		claims.mu.Lock()
		defer claims.mu.Unlock()
		delete(claims.held, key)
	})
	return true
}

// entryOf returns the entry that keeps e, a failing case of the property
// contract, found at found.
func entryOf(contract string, e engine.Execution, found time.Time) store.Entry {
	return store.Entry{
		Definition:     definition,
		Property:       contract,
		Identity:       identityOf(e.Identity),
		Choices:        e.Case.Choices(),
		Counterexample: drawsOf(e.Case.Draws()),
		Found:          found,
	}
}

// identityOf returns a failure identity in the form of an entry: the base
// name of its file, and its panic's type as an error.
func identityOf(i engine.Identity) store.Identity {
	id := store.Identity{Assertion: i.Assertion, Contract: i.Contract, Error: i.Panic}
	if i.Where != (assert.Where{}) {
		id.File, id.Line = filepath.Base(i.Where.File), i.Where.Line
	}
	return id
}

// drawsOf returns the draws of a case in the form of an entry: each label
// with the typed literal of its value, and without a value where no typed
// literal states it.
func drawsOf(draws []engine.Drawn) []store.Draw {
	out := make([]store.Draw, len(draws))
	for i, d := range draws {
		out[i].Label = d.Label
		if value, ok := store.Literal(d.Value); ok {
			out[i].Value = value
		}
	}
	return out
}

// differences returns a note for each stored entry whose case now decodes
// to other values than the entry records. runs are the runs of the stored
// cases in the order of entries, up to the one that ended the run.
func differences(entries []store.Entry, runs []engine.Execution) []string {
	var notes []string
	for i, run := range runs {
		if !alike(entries[i].Counterexample, drawsOf(run.Case.Draws())) {
			notes = append(notes, fmt.Sprintf(
				"prop: the stored case %s decodes to other values than it records, and the run tested those",
				entries[i].Name()))
		}
	}
	return notes
}

// alike reports whether replayed states the labels of recorded in order,
// and the value of each draw whose value recorded states. Two values are
// alike when they decode to the same JSON value, so that the spelling of a
// number or an escape in a file that another implementation wrote does not
// count.
func alike(recorded, replayed []store.Draw) bool {
	if len(recorded) != len(replayed) {
		return false
	}
	for i, r := range recorded {
		if r.Label != replayed[i].Label || (r.Value != nil && !sameValue(r.Value, replayed[i].Value)) {
			return false
		}
	}
	return true
}

// sameValue reports whether a and b are JSON texts of the same value. A
// stored value a is JSON, because its file parsed. A replayed value b is
// nil for a value that no typed literal states, which is the same as no
// stored value.
func sameValue(a, b json.RawMessage) bool {
	var x, y any
	return b != nil && json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && reflect.DeepEqual(x, y)
}
