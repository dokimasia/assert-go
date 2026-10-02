// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

// propGlob matches the vector files of the property engine in the vendored
// definition. The corpus glob of the assertions does not reach them.
const propGlob = "spec/corpus/prop/*.json"

// VectorKind is the kind of a vector of the definition's property engine.
// Its spelling is the name of the file that states the vectors of the kind.
type VectorKind string

const (
	// Decoding decodes a generator from stated choices.
	Decoding VectorKind = "decoding"
	// Generation decodes a generator from the first cases of a seed.
	Generation VectorKind = "generation"
	// Shrinking runs a property of one draw and shrinks its failure.
	Shrinking VectorKind = "shrinking"
	// Coverage decides one coverage requirement at one check.
	Coverage VectorKind = "coverage"
	// Bridge decodes a generator from a fuzzer's bytes.
	Bridge VectorKind = "bridge"
	// Token encodes choices as a replay token, or decodes one.
	Token VectorKind = "token"
	// Behaviour runs a body under settings and states the record.
	Behaviour VectorKind = "behaviour"
	// Store writes a store entry, or reads the text of one file.
	Store VectorKind = "store"
)

// Valid reports whether k is one of the eight kinds. It allocates nothing.
func (k VectorKind) Valid() bool {
	switch k {
	case Decoding, Generation, Shrinking, Coverage, Bridge, Token, Behaviour, Store:
		return true
	}
	return false
}

// Vector is one case of a vector file: its kind, its id, and the inputs and
// the outputs that its kind states. A test builds a Vector of its own to
// drive a rule that the definition's vectors cannot, such as the refusal
// of a vector that misstates its outputs.
//
// # Concurrency
//
// [Vector.Check] reads a Vector and changes nothing in it. Checks run
// concurrently when each has a directory of its own.
type Vector struct {
	// Kind is the vector's kind.
	Kind VectorKind
	// ID names the vector within the corpus.
	ID string
	// Raw is the case's JSON object, as its file states it.
	Raw json.RawMessage
}

// Vectors returns every vector of the vendored definition, the files in
// the order of their names and each file's cases in order. A vector takes
// the kind that its file states, and [Vector.Check] refuses a kind outside
// the eight.
//
// It returns an error for a file that does not read or parse.
//
// # Allocation contract
//
// Vectors allocates 483 times on the vendored definition: the names that
// the glob returns, the open file and the copy of each of the eight files,
// the two structs that each file decodes into with their lists of cases, a
// copy of each case's JSON, and each case's id. The JSON decoder's pooled
// state, which a garbage collection or a move of the goroutine to another
// processor leaves empty, adds up to two.
func Vectors() ([]Vector, error) { return vectorsIn(definition, propGlob) }

// vectorsIn returns the vectors of the files of fsys that glob matches.
func vectorsIn(fsys fs.FS, glob string) ([]Vector, error) {
	names, err := fs.Glob(fsys, glob)
	if err != nil {
		return nil, fmt.Errorf("conformance: glob the vectors: %w", err)
	}
	var out []Vector
	for _, name := range names {
		raw, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("conformance: read %s: %w", name, err)
		}
		vectors, err := vectorsOf(raw)
		if err != nil {
			return nil, fmt.Errorf("conformance: %s: %w", name, err)
		}
		out = append(out, vectors...)
	}
	return out, nil
}

// vectorsOf returns the vectors of one file's JSON. It decodes the file
// twice: once for the JSON of each case, and once for each case's id.
func vectorsOf(raw []byte) ([]Vector, error) {
	var file struct {
		Kind  VectorKind        `json:"kind"`
		Cases []json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("conformance: parse vectors: %w", err)
	}
	var heads struct {
		Cases []struct {
			ID string `json:"id"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &heads); err != nil {
		return nil, fmt.Errorf("conformance: parse the ids of the vectors: %w", err)
	}
	out := make([]Vector, len(file.Cases))
	for i, c := range file.Cases {
		out[i] = Vector{Kind: file.Kind, ID: heads.Cases[i].ID, Raw: c}
	}
	return out, nil
}

// Check runs v against this implementation. It returns how the outputs
// differ from the ones that v states, or nil when they match. A behaviour
// vector writes its stored cases to dir, an empty directory.
//
// It returns an error for a vector whose inputs do not parse, and for one
// of a kind outside the eight.
//
// # Allocation contract
//
// Check allocates the struct that the vector's JSON decodes into, and the
// values of what its kind runs. A token vector of no choices allocates
// twice: the struct and the token. A behaviour vector allocates a whole run
// of [go.dokimi.dev/assert/prop.ForAll].
func (v Vector) Check(dir string) error {
	var err error
	switch v.Kind {
	case Decoding:
		err = checkDecoding(v.Raw)
	case Generation:
		err = checkGeneration(v.Raw)
	case Shrinking:
		err = checkShrinking(v.Raw)
	case Coverage:
		err = checkCoverage(v.Raw)
	case Bridge:
		err = checkBridge(v.Raw)
	case Token:
		err = checkToken(v.Raw)
	case Behaviour:
		err = checkBehaviour(v.Raw, dir)
	case Store:
		err = checkStore(v.Raw)
	default:
		err = fmt.Errorf("%q is no vector kind", v.Kind)
	}
	if err != nil {
		return fmt.Errorf("conformance: %s: %w", v.ID, err)
	}
	return nil
}

// sameValue reports whether got is the value that the typed literal want
// states.
func sameValue(got any, want json.RawMessage) (bool, error) {
	w, err := Decode(want)
	if err != nil {
		return false, err
	}
	return canonical(got) == canonical(w), nil
}

// decodedCase is the value that one case decoded, as a vector states it:
// a typed literal, null for a rejected case, and whether the case was
// rejected.
type decodedCase struct {
	// Value is the typed literal of the value.
	Value json.RawMessage `json:"value"`
	// Rejected reports whether the case was rejected.
	Rejected bool `json:"rejected"`
}

// compare returns how the outcome of e, which drew got, differs from the
// one that d states with the choices recorded, or nil when they match.
func (d decodedCase) compare(e engineOutcome, got any, recorded []json.RawMessage) error {
	if e.rejected != d.Rejected {
		return fmt.Errorf("rejected is %t, want %t", e.rejected, d.Rejected)
	}
	same, err := sameChoices(e.choices, recorded)
	if err != nil {
		return err
	}
	if !same {
		return fmt.Errorf("the case records %v, want %s", e.choices, jsonOf(recorded))
	}
	if d.Rejected {
		return nil
	}
	same, err = sameValue(got, d.Value)
	if err != nil {
		return err
	}
	if !same {
		return fmt.Errorf("the value is %s, want %s", canonical(got), d.Value)
	}
	return nil
}

// jsonOf returns the JSON text of v, for a message.
func jsonOf(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}
