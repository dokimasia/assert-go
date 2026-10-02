// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/conformance"
)

// vectorCount is the number of vectors of the vendored definition: 26
// behaviour, 11 bridge, 12 coverage, 49 decoding, 32 generation, 36
// shrinking, 21 store and 17 token vectors.
const vectorCount = 204

// The first vector of the definition, and the prefix of every error that
// Check returns for a vector that a test builds.
const (
	// firstVector is the id of the first case of behaviour.json, the file
	// whose name sorts first.
	firstVector = "behaviour/passes-a-body-that-never-fails"
	// builtID is the id of a vector that a test builds.
	builtID = "built-by-the-test"
	// builtPrefix starts the text of an error of a built vector.
	builtPrefix = "conformance: " + builtID + ": "
)

// emptyToken is a token vector that encodes no choices, the cheapest
// vector to check.
const emptyToken = `{"choices":[],"token":"prop1:"}`

// The allocations of the functions of a vector, measured once the JSON
// decoder has cached the functions of their types.
const (
	// validAllocs are the allocations of Valid.
	validAllocs = 0
	// vectorsAllocs are the allocations of Vectors: the 483 of its contract,
	// and the two that the JSON decoder's pooled state adds.
	vectorsAllocs = 485
	// checkAllocs are the allocations of Check on emptyToken: the struct
	// that the vector decodes into, and the token that Encode returns.
	checkAllocs = 2
)

// TestVector runs every vector of the property engine against this
// implementation, and drives the rules of a vector that the definition's
// vectors cannot. Written with testing rather than with this library,
// because a verdict is not written with the subject.
func TestVector(t *testing.T) {
	t.Parallel()

	vectors, err := conformance.Vectors()
	if err != nil {
		t.Fatalf("the vectors read: %v", err)
	}

	t.Run("Valid", func(t *testing.T) {
		t.Parallel()

		t.Run("reports true for each of the eight kinds", func(t *testing.T) {
			t.Parallel()
			kinds := []conformance.VectorKind{
				conformance.Decoding, conformance.Generation, conformance.Shrinking, conformance.Coverage,
				conformance.Bridge, conformance.Token, conformance.Behaviour, conformance.Store,
			}
			for _, k := range kinds {
				if !k.Valid() {
					t.Fatalf("Valid of %q reports false", k)
				}
			}
		})

		tests := []struct {
			name string
			give conformance.VectorKind
		}{
			{name: "reports false for the empty kind", give: ""},
			{name: "reports false for a kind that names no file of the definition", give: "fuzzing"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				if tt.give.Valid() {
					t.Fatalf("Valid of %q reports true", tt.give)
				}
			})
		}
	})

	t.Run("Vectors", func(t *testing.T) {
		t.Parallel()

		t.Run("returns every vector of the definition", func(t *testing.T) {
			t.Parallel()
			if len(vectors) != vectorCount {
				t.Fatalf("read %d vectors, want %d", len(vectors), vectorCount)
			}
		})

		t.Run("returns the files in the order of their names", func(t *testing.T) {
			t.Parallel()
			byKind := func(a, b conformance.Vector) int { return strings.Compare(string(a.Kind), string(b.Kind)) }
			if !slices.IsSortedFunc(vectors, byKind) {
				t.Fatal("the vectors are not in the order of their files' names")
			}
		})

		t.Run("returns the cases of a file in the order the file states them", func(t *testing.T) {
			t.Parallel()
			if vectors[0].ID != firstVector {
				t.Fatalf("the first vector is %s, want %s", vectors[0].ID, firstVector)
			}
		})
	})

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		for _, v := range vectors {
			t.Run("returns nil for "+string(v.Kind)+" vector "+v.ID, func(t *testing.T) {
				t.Parallel()
				if err := v.Check(t.TempDir()); err != nil {
					t.Fatal(err)
				}
			})
		}

		t.Run("returns an error that starts with the vector's id", func(t *testing.T) {
			t.Parallel()
			err := check(t, conformance.Token, `{"choices":[],"token":"prop1:AAA"}`)
			if err == nil || !strings.HasPrefix(err.Error(), builtPrefix) {
				t.Fatalf("Check returns %v, want an error that starts with %q", err, builtPrefix)
			}
		})

		tests := []struct {
			name string
			kind conformance.VectorKind
			give string
			want string
		}{
			{
				name: "returns an error for a vector of a kind outside the eight",
				kind: "fuzzing",
				give: emptyToken,
				want: `"fuzzing" is no vector kind`,
			},
			{
				name: "returns an error for a decoded case that the vector states as rejected",
				kind: conformance.Decoding,
				give: decoded(digitGenerator, `[7]`, `[7]`, null),
				want: "rejected is false, want true",
			},
			{
				name: "returns an error for a rejected case that the vector states as decoded",
				kind: conformance.Decoding,
				give: decoded(rejectingGenerator, `[]`, `[]`, `{"type":"int","value":1}`),
				want: "rejected is true, want false",
			},
			{
				name: "returns an error for a value of another type than the literal states",
				kind: conformance.Decoding,
				give: decoded(digitGenerator, `[7]`, `[7]`, `{"type":"string","value":"7"}`),
				want: "the value is int:7",
			},
			{
				name: "returns an error for a value literal of an unknown type",
				kind: conformance.Decoding,
				give: decoded(digitGenerator, `[7]`, `[7]`, `{"type":"widget"}`),
				want: conformance.ErrUnknownType.Error(),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, tt.kind, tt.give), tt.want)
			})
		}
	})
}

// TestVectorZeroAlloc checks the allocation ceilings of the functions of a
// vector.
func TestVectorZeroAlloc(t *testing.T) {
	v := conformance.Vector{Kind: conformance.Token, ID: builtID, Raw: json.RawMessage(emptyToken)}
	dir := t.TempDir()
	assert.MaxAllocs(t, func() { _ = conformance.Token.Valid() }, validAllocs, "Valid allocates nothing")
	assert.MaxAllocs(t, func() { _, _ = conformance.Vectors() }, vectorsAllocs, "Vectors allocates its files and cases")
	assert.MaxAllocs(t, func() { _ = v.Check(dir) }, checkAllocs, "Check allocates the vector's fields and its token")
}

// BenchmarkVector measures each function of a vector. A benchmark of a
// function that decodes JSON calls it once before the measured loop, which
// then measures the decoder with the functions of its types cached.
func BenchmarkVector(b *testing.B) {
	b.Run("Valid", func(b *testing.B) {
		var got bool
		c := bench.Start(b).MaxAllocs(validAllocs)
		defer c.End()
		for c.Loop() {
			got = conformance.Token.Valid()
		}
		assert.True(b, got, "token is one of the eight kinds")
	})

	b.Run("Vectors", func(b *testing.B) {
		got, _ := conformance.Vectors()
		c := bench.Start(b).MaxAllocs(vectorsAllocs)
		defer c.End()
		for c.Loop() {
			got, _ = conformance.Vectors()
		}
		assert.Length(b, got, vectorCount, "every vector of the definition")
	})

	b.Run("Check", func(b *testing.B) {
		v := conformance.Vector{Kind: conformance.Token, ID: builtID, Raw: json.RawMessage(emptyToken)}
		dir := b.TempDir()
		got := v.Check(dir)
		c := bench.Start(b).MaxAllocs(checkAllocs)
		defer c.End()
		for c.Loop() {
			got = v.Check(dir)
		}
		assert.NoError(b, got, "the token of no choices is the prefix")
	})
}

// check returns what Check returns for a vector of kind whose case is the
// JSON text raw, with a directory of its own for stored cases.
func check(t *testing.T, kind conformance.VectorKind, raw string) error {
	t.Helper()
	return conformance.Vector{Kind: kind, ID: builtID, Raw: json.RawMessage(raw)}.Check(t.TempDir())
}

// expectCheck fails t unless err is nil for an empty want, or an error
// whose text contains want.
func expectCheck(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("Check returns %v, want nil", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Check returns %v, want an error that contains %q", err, want)
	}
}
