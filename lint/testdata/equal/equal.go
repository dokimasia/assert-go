// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package equal

import (
	"bytes"
	"encoding/json"
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type kind int

const kindA kind = 1

type point struct{ x, y int }

type list struct {
	next  *list
	value int
}

type ratio struct{ value float64 }

type handler struct{ serve func() }

type ids []int

func correct(t *testing.T, n int, xs []int) {
	assert.Equal(t, n, 3, "the count is three")
	expect.Equal(t, xs, []int{1, 2}, "the store returns two items", expect.EquateEmpty())
}

func compared(t *testing.T, n int, s string, ok bool, k kind, d time.Duration) {
	assert.True(t, n == 3, "the count is three")              // want `compare: state the check with Equal`
	assert.True(t, 3 == n, "the count is three")              // want `compare: state the check with Equal`
	expect.True(t, ok == true, "the flag is set")             // want `compare: state the check with Equal`
	assert.True(t, k == kindA, "the kind is A")               // want `compare: state the check with Equal`
	assert.True(t, d == time.Second, "the delay is a second") // want `compare: state the check with Equal`
	if s != "x" {                                             // want `compare: state the check with Equal`
		t.Fatalf("got %q, want x", s)
	}
}

func deep(t *testing.T, a, b []int, p, q *list, m, mm map[string][]int, r, s ratio, h, g handler) {
	assert.True(t, reflect.DeepEqual(a, b), "the stores hold one list")  // want `deep-equal: state the check with Equal`
	expect.True(t, reflect.DeepEqual(p, q), "the lists are equal")       // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(m, mm), "the indexes are equal")    // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(r, s), "the ratios are equal")      // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(h, g), "the handlers are equal")    // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(a, nil), "the store holds no list") // want `deep-equal: state the check with Equal`
	if !reflect.DeepEqual(a, b) {                                        // want `deep-equal: state the check with Equal`
		t.Errorf("got %v, want %v", a, b)
	}
}

func keyed(t *testing.T, byAddress, other map[*int]int, keys, others map[point]int, x, y any, c, d chan int, e, f [2]string) {
	assert.True(t, reflect.DeepEqual(byAddress, other), "the counts are equal") // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(keys, others), "the counts are equal")     // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(x, y), "the values are equal")             // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(c, d), "the channels are one")             // want `deep-equal: state the check with Equal`
	assert.True(t, reflect.DeepEqual(e, f), "the pairs are equal")              // want `deep-equal: state the check with Equal`
}

func collections(t *testing.T, a, b []byte, xs, ys ids, raw json.RawMessage, m, mm map[string]int, ps, qs []point, pm, qm map[string]point) {
	assert.True(t, bytes.Equal(a, b), "the bodies match")         // want `equal-func: state the check with Equal and EquateEmpty`
	expect.True(t, slices.Equal(xs, ys), "the identifiers match") // want `equal-func: state the check with Equal and EquateEmpty`
	assert.True(t, maps.Equal(m, mm), "the counts match")         // want `equal-func: state the check with Equal and EquateEmpty`
	assert.True(t, bytes.Equal(raw, a), "the raw body matches")   // want `equal-func: state the check with Equal and EquateEmpty`
	assert.True(t, slices.Equal(ps, qs), "the points match")
	assert.True(t, maps.Equal(pm, qm), "the points match")
	if !bytes.Equal(a, b) { // want `equal-func: state the check with Equal and EquateEmpty`
		t.Fatal("the bodies differ")
	}
}

func generic[S ~[]int](t *testing.T, a, b S) {
	assert.True(t, slices.Equal(a, b), "the lists match")
}
