// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package permutation

import (
	"slices"
	"sort"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func list() []string { return nil }

func correct(t *testing.T, got, want []string) {
	assert.Permutation(t, got, want, "the store returns every name once")
}

func sorted(t *testing.T, want []string, ids, wantIDs []int) {
	got := list()
	slices.Sort(got)
	slices.Sort(want)
	assert.Equal(t, got, want, "the store returns every name once") // want `permutation: state the check with Permutation of slices\.Sort\(got\) and slices\.Sort\(want\)`
	names := list()
	sort.Strings(names)
	t.Log(len(names))
	expect.Equal(t, want, names, "the store returns every name once") // want `permutation: state the check with Permutation`
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	sort.Ints(wantIDs)
	if !slices.Equal(ids, wantIDs) { // want `permutation: state the check with Permutation of sort\.Slice\(…\) and sort\.Ints\(wantIDs\)`
		t.Fatalf("got %v", ids)
	}
}

func ordered(t *testing.T, want []string) {
	got := list()
	slices.Sort(got)
	assert.Equal(t, got, want, "the store returns the names in order")
	expected := []string{"b", "a"}
	slices.Sort(expected)
	assert.Equal(t, list(), expected, "the store returns the names in order")
	other := list()
	slices.Sort(other)
	assert.NotEqual(t, other, want, "the sorted names differ")
}
