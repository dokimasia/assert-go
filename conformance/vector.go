// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance

import (
	"encoding/json"
	"io/fs"
	"path"
	"slices"
	"strings"

	"go.dokimi.dev/assert/files"
	"go.dokimi.dev/assert/internal/fault"
)

// vectorGlobs match the vector files of the vendored definition: those of
// the assertions that read files, of the history, of the property engine and
// of machines. The corpus glob of the assertions matches none of them.
var vectorGlobs = [...]string{
	"spec/corpus/files/*.json", "spec/corpus/history/*.json", "spec/corpus/prop/*.json", "spec/corpus/stateful/*.json",
}

// The members of a corpus case, of a vector and of the specs inside one,
// that the path of a fault names.
const (
	// Of a corpus case and of a vector.
	expectMember    = "expect"
	argsMember      = "args"
	bytesMember     = "bytes"
	callsMember     = "calls"
	casesMember     = "cases"
	choicesMember   = "choices"
	decodedMember   = "decoded"
	detailMember    = "detail"
	drawsMember     = "draws"
	entriesMember   = "entries"
	errorMember     = "error"
	examplesMember  = "examples"
	failsWhenMember = "fails-when"
	fixtureMember   = "fixture"
	formMember      = "form"
	foundMember     = "found"
	generatorMember = "generator"
	labelMember     = "label"
	nameMember      = "name"
	outcomeMember   = "outcome"
	recordedMember  = "recorded"
	rejectedMember  = "rejected"
	runsMember      = "runs"
	seedMember      = "seed"
	settingsMember  = "settings"
	shapeMember     = "shape"
	storedMember    = "stored"
	subjectMember   = "subject"
	subjectsMember  = "subjects"
	tokenMember     = "token"
	valueMember     = "value"
	valuesMember    = "values"
	verdictMember   = "verdict"
	entryMember     = "entry"
	bodyMember      = "body"
	replayMember    = "replay"

	// Of a generator spec.
	genMember     = "gen"
	minMember     = "min"
	maxMember     = "max"
	widthMember   = "width"
	pMember       = "p"
	ofMember      = "of"
	keysMember    = "keys"
	patternMember = "pattern"
	baseMember    = "base"
	extendMember  = "extend"
	keepMember    = "keep"

	// Of a predicate spec, a body spec, a choice and bounds.
	kindMember        = "kind"
	nMember           = "n"
	drawMember        = "draw"
	classifyMember    = "classify"
	rejectsWhenMember = "rejects-when"
	failsMember       = "fails"
	whenMember        = "when"
	floatMember       = "float"
)

// VectorKind is the kind of a vector of the definition: of the property
// engine, of the history and its checker, of machines, or of an assertion
// that reads files. Its spelling is the name of the file that states the
// vectors of the kind.
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
	// Shapes decodes a shape from the first cases of a seed.
	Shapes VectorKind = "shapes"
	// Inverse runs a shape or a generator backwards from a value.
	Inverse VectorKind = "inverse"
	// Draws runs a body under a case of Draws entries.
	Draws VectorKind = "draws"
	// Fixtures reads a fixture type into its shape.
	Fixtures VectorKind = "fixtures"
	// Forms runs a property form on built subjects.
	Forms VectorKind = "forms"
	// CallRecords runs a body under settings and states the call records of
	// the run.
	CallRecords VectorKind = "recording"
	// Seam records a script or intervals through the history, and states the
	// events or the entry that the history refuses.
	Seam VectorKind = "seam"
	// Linearizable checks a history against a named spec, and states the
	// verdict and the detail of its record.
	Linearizable VectorKind = "linearizable"
	// Serializable checks a history of list-append transactions for the
	// anomalies that serializability forbids, and states the verdict and the
	// detail of its record.
	Serializable VectorKind = "serializable"
	// SnapshotIsolation checks a history of list-append transactions for the
	// anomalies that snapshot isolation forbids, and states the verdict and
	// the detail of its record.
	SnapshotIsolation VectorKind = "snapshot-isolation"
	// Machines runs the steps of a named machine subject under a setup,
	// settings and a trace, and states the detail of the run or the refusal
	// of the trace.
	Machines VectorKind = "machines"
	// TreeEqual compares the tree of a workspace with a stated tree through
	// files.Equal, and states the verdict and the detail of its record.
	TreeEqual VectorKind = "tree-equal"
	// TreeContains compares the tree of a workspace with a stated tree
	// through files.Contains, and states the verdict and the detail of its
	// record.
	TreeContains VectorKind = "tree-contains"
	// TreeUnchanged calls a subject on the tree of a workspace through
	// files.Unchanged, and states the verdict and the detail of its record.
	TreeUnchanged VectorKind = "tree-unchanged"
	// GoldenMatchTree compares the tree of a workspace with a golden tree
	// through golden.MatchTree, and states the verdict, the detail of its
	// record, and the golden tree that an update leaves.
	GoldenMatchTree VectorKind = "golden-match-tree"
	// PathAbsent checks a path of a workspace through files.Absent, and
	// states the verdict and the detail of its record.
	PathAbsent VectorKind = "path-absent"
	// IsFile checks a path of a workspace through files.IsFile, and states
	// the verdict and the detail of its record.
	IsFile VectorKind = "is-file"
	// IsDir checks a path of a workspace through files.IsDir, and states the
	// verdict and the detail of its record.
	IsDir VectorKind = "is-dir"
	// LinksTo checks a path of a workspace through files.LinksTo, and states
	// the verdict and the detail of its record.
	LinksTo VectorKind = "links-to"
	// HasContent checks a path of a workspace through files.HasContent, and
	// states the verdict and the detail of its record.
	HasContent VectorKind = "has-content"
	// HasMode checks a path of a workspace through files.HasMode, and states
	// the verdict and the detail of its record.
	HasMode VectorKind = "has-mode"
)

// runner runs the JSON of one vector against this implementation, and
// returns a fault of how the outputs differ from the ones that the vector
// states, with a path inside the vector. A behaviour vector, a recording
// vector and a machines vector write their stored cases to dir, a files
// vector its workspace, and a golden-match-tree vector its golden tree.
type runner func(raw json.RawMessage, dir string) error

// runners maps each of the twenty-nine kinds to its runner.
var runners = map[VectorKind]runner{
	Decoding:          checkDecoding,
	Generation:        checkGeneration,
	Shrinking:         checkShrinking,
	Coverage:          checkCoverage,
	Bridge:            checkBridge,
	Token:             checkToken,
	Behaviour:         checkBehaviour,
	Store:             checkStore,
	Shapes:            checkShapes,
	Inverse:           checkInverse,
	Draws:             checkDraws,
	Fixtures:          checkFixtures,
	Forms:             checkForms,
	CallRecords:       checkRecording,
	Seam:              checkSeam,
	Linearizable:      checkLinearizable,
	Serializable:      checkSerializable,
	SnapshotIsolation: checkSnapshotIsolation,
	Machines:          checkMachines,
	TreeEqual:         checkFiles(treeCall(files.Equal)),
	TreeContains:      checkFiles(treeCall(files.Contains)),
	TreeUnchanged:     checkFiles(unchangedCall),
	GoldenMatchTree:   checkFiles(goldenCall),
	PathAbsent:        checkFiles(kindCall(files.Absent)),
	IsFile:            checkFiles(kindCall(files.IsFile)),
	IsDir:             checkFiles(kindCall(files.IsDir)),
	LinksTo:           checkFiles(linksToCall),
	HasContent:        checkFiles(hasContentCall),
	HasMode:           checkFiles(hasModeCall),
}

// Valid reports whether k is one of the twenty-nine kinds. It allocates
// nothing.
func (k VectorKind) Valid() bool {
	_, ok := runners[k]
	return ok
}

// Vector is one case of a vector file: its kind, its id, and the inputs and
// the outputs that its kind states. A test builds a Vector of its own to
// drive a rule that the definition's vectors cannot, such as the refusal
// of a vector that misstates its outputs.
//
// # Concurrency
//
// [Vector.Check] reads a Vector and changes nothing in it. Checks run
// concurrently when each has a directory of its own, except the checks of
// golden-match-tree vectors, whose directory is the working directory of the
// process.
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
// the twenty-nine.
//
// # Allocation contract
//
// Vectors allocates 1,609 times on the vendored definition: the names that
// the four globs return, the open file and the copy of each of the
// twenty-nine files, the two structs that each file decodes into with their
// lists of cases, a copy of each case's JSON, each case's id, and the growth
// of the list that it returns. The JSON decoder's pooled state, which a
// garbage collection or a move of the goroutine to another processor leaves
// empty, adds up to two.
func Vectors() []Vector {
	var names []string
	for _, pattern := range vectorGlobs {
		// The pattern is well formed, so Glob returns no error.
		matches, _ := fs.Glob(definition, pattern)
		names = append(names, matches...)
	}
	slices.SortFunc(names, func(a, b string) int { return strings.Compare(path.Base(a), path.Base(b)) })
	var out []Vector //nolint:prealloc // the number of cases is known only after each file is read
	for _, name := range names {
		var file struct {
			Kind  VectorKind        `json:"kind"`
			Cases []json.RawMessage `json:"cases"`
		}
		var heads struct {
			Cases []struct {
				ID string `json:"id"`
			} `json:"cases"`
		}
		read(name, &file, &heads)
		for i, c := range file.Cases {
			out = append(out, Vector{Kind: file.Kind, ID: heads.Cases[i].ID, Raw: c})
		}
	}
	return out
}

// Check runs v against this implementation. It returns how the outputs
// differ from the ones that v states, or nil when they match. A behaviour
// vector, a recording vector and a machines vector write their stored cases
// to dir, an empty directory, and a files vector writes its workspace there.
// golden.MatchTree resolves the name of a golden tree against the working
// directory of the process, so a golden-match-tree vector writes its golden
// tree below dir, which must be the working directory.
//
// # Errors
//
// It returns a fault whose path starts at the vector's ID and leads through
// the vector's JSON to the part at fault. That part is an input that does
// not parse or that the vocabulary does not state, or an output that
// differs from the run. A vector of a kind outside the twenty-nine has a
// fault at its ID alone, and so does a golden-match-tree vector whose dir is
// not the working directory.
//
// # Allocation contract
//
// Check allocates the struct that the vector's JSON decodes into, and the
// values of what its kind runs. A token vector of no choices allocates
// twice: the struct and the token. A behaviour vector, a recording vector
// and a machines vector allocate a whole run of
// [go.dokimi.dev/assert/prop.ForAll], and a forms vector a whole run of its
// form.
func (v Vector) Check(dir string) error {
	run, ok := runners[v.Kind]
	if !ok {
		return fault.At(fault.New("%q is no vector kind", v.Kind), fault.Field(v.ID))
	}
	if err := run(v.Raw, dir); err != nil {
		return fault.At(err, fault.Field(v.ID))
	}
	return nil
}
