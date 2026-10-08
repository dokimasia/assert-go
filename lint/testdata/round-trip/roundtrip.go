// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package roundtrip

import (
	"encoding/json"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

type order struct {
	ID    int
	Stamp string `json:"-"`
}

func encode(n int) string { return strconv.Itoa(n) }

func decode(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func marshal(o order) ([]byte, error) { return json.Marshal(o) }

func unmarshal(data []byte) (order, error) {
	var o order
	err := json.Unmarshal(data, &o)
	return o, err
}

func correct(t *testing.T, v order) {
	assert.RoundTrip(t, marshal, unmarshal, v, "decoding undoes encoding")
}

func converted(t *testing.T, v order, n int) {
	data, err := json.Marshal(v)
	assert.NoError(t, err, "the order encodes")
	var got order
	assert.NoError(t, json.Unmarshal(data, &got), "the order decodes")
	assert.Equal(t, got, v, "decoding undoes encoding")                // want `round-trip: state the check with RoundTrip`
	expect.Equal(t, n, decode(encode(n)), "parsing undoes formatting") // want `round-trip: state the check with RoundTrip of encode and decode`
	format := encode
	expect.Equal(t, n, decode(format(n)), "parsing undoes formatting") // want `round-trip: state the check with RoundTrip`
}

func stepped(t *testing.T, n int) {
	text := encode(n)
	back := decode(text)
	if back != n { // want `round-trip: state the check with RoundTrip`
		t.Fatalf("got %d, want %d", back, n)
	}
}

func unrelated(t *testing.T, n int) {
	assert.Equal(t, len(encode(n)), n, "the text has n digits")
	assert.Equal(t, decode(encode(3)), 3, "the text of three parses back")
	text := encode(n)
	assert.Equal(t, encode(len(text)), text, "the length is no conversion")
}

func unordered(t *testing.T, n int) {
	var later int
	run := func() { later = decode(encode(n)) }
	run()
	assert.Equal(t, later, n, "the closure decodes")
	var looped int
	for range 2 {
		looped = decode(encode(n))
	}
	assert.Equal(t, looped, n, "the loop decodes")
}

func interrupted(t *testing.T, v order, key []byte) {
	data, err := marshal(v)
	assert.NoError(t, err, "the order encodes")
	clear(key)
	got, err := unmarshal(data)
	assert.NoError(t, err, "a cleared key changes no decoding")
	assert.Equal(t, got, v, "decoding undoes encoding")
}

func touched(t *testing.T, v order) {
	data, err := marshal(v)
	assert.NoError(t, err, "the order encodes")
	got, err := unmarshal(data)
	assert.NoError(t, err, "the order decodes")
	got.Stamp = v.Stamp
	assert.Equal(t, got, v, "decoding undoes encoding but for the stamp")
}

func inspected(t *testing.T, v order) {
	data, err := marshal(v)
	assert.NoError(t, err, "the order encodes")
	assert.NotEmpty(t, data, "the encoding has bytes")
	got, err := unmarshal(data)
	assert.NoError(t, err, "the order decodes")
	assert.Equal(t, got, v, "decoding undoes encoding")
}

func kept(t *testing.T, v order) []byte {
	data, err := marshal(v)
	assert.NoError(t, err, "the order encodes")
	got, err := unmarshal(data)
	assert.NoError(t, err, "the order decodes")
	assert.Equal(t, got, v, "decoding undoes encoding")
	return data
}
