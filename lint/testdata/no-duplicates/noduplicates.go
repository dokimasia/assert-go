// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package noduplicates

import (
	"testing"

	"go.dokimi.dev/assert"
)

func list() ([]string, error) { return nil, nil }

func correct(t *testing.T) {
	assert.NoDuplicates(t, list, "the store lists each id once")
}

func seen(t *testing.T, ids []string, words []string) {
	seen := make(map[string]bool)
	for _, id := range ids { // want `no-duplicates: state the check with NoDuplicates`
		assert.False(t, seen[id], "the store lists each id once")
		seen[id] = true
	}
	found := make(map[string]struct{})
	for i := 0; i < len(ids); i++ { // want `no-duplicates: state the check with NoDuplicates`
		if _, dup := found[ids[i]]; dup { // want `contains: state the check with NotContains`
			t.Fatalf("the id %s repeats", ids[i])
		}
		found[ids[i]] = struct{}{}
	}
	counts := make(map[string]int)
	for _, w := range words {
		counts[w]++
		t.Log(w)
	}
	total := make(map[string]int)
	for _, w := range words {
		total[w] = total[w] + 1
		assert.NotEmpty(t, w, "every word is set")
	}
}
