// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/filetree"
)

// builtID is the id of a vector or a case that a test builds.
const builtID = "built-by-the-test"

// kinds are the twenty-nine kinds of a vector.
var kinds = []conformance.VectorKind{
	conformance.Decoding, conformance.Generation, conformance.Shrinking, conformance.Coverage,
	conformance.Bridge, conformance.Token, conformance.Behaviour, conformance.Store, conformance.Shapes,
	conformance.Inverse, conformance.Draws, conformance.Fixtures, conformance.Forms, conformance.CallRecords,
	conformance.Seam, conformance.Linearizable, conformance.Serializable, conformance.SnapshotIsolation,
	conformance.Machines, conformance.TreeEqual, conformance.TreeContains, conformance.TreeUnchanged,
	conformance.GoldenMatchTree, conformance.PathAbsent, conformance.IsFile, conformance.IsDir,
	conformance.LinksTo, conformance.HasContent, conformance.HasMode,
}

// The generators and the literals that the tests of several kinds build
// their vectors from.
const (
	// digitGenerator is the integer generator over [0, 9].
	digitGenerator = `{"gen":"integer","min":0,"max":9}`
	// unknownGenerator is a generator spec of an id outside the vocabulary.
	unknownGenerator = `{"gen":"widget"}`
	// widget is a typed literal of a type that the encoding does not define.
	widget = `{"type":"widget"}`
	// oneLiteral is the typed literal of the integer 1.
	oneLiteral = `{"type":"int","value":1}`
	// four is the typed literal of the integer 4.
	four = `{"type":"int","value":4}`
	// null is the JSON text of no value.
	null = "null"
	// nullLiteral is the typed literal of null.
	nullLiteral = `{"type":"null"}`
)

// The reasons of the faults of a vector's inputs that several tests state.
const (
	// notParsed is the reason of a vector that does not parse.
	notParsed = "the vector does not parse"
	// unknownWidget is the reason of the fault of the literal widget, at
	// the literal's member type.
	unknownWidget = `the type "widget" is no type of the encoding`
	// noGenerator is the reason of the fault of unknownGenerator, at its
	// member gen.
	noGenerator = `"widget" names no generator`
)

// The settings, the body and the detail of a behaviour vector's run that
// several tests build.
const (
	// seven are the settings of a run of seed 7.
	seven = `{"seed":"7"}`
	// passingBody draws an integer in [0, 1000] and never fails.
	passingBody = `{"draw":{"gen":"integer","min":0,"max":1000}}`
	// passedDetail is the detail of the run of passingBody under seven.
	passedDetail = `{"outcome":"passed","cases":100,"rejected":0,"seed":"7","counterexample":null,` +
		`"failure":null,"choices":null,"others":null,"divergence":null,"coverage":null}`
	// bigBody draws an integer in [0, 10000] and fails as big from 1001.
	bigBody = `{"draw":{"gen":"integer","min":0,"max":10000},` +
		`"fails":[{"identity":"big","when":{"kind":"at-least","n":1001}}]}`
	// bigDraw is the draw of the minimal counterexample of bigBody.
	bigDraw = `{"label":"value","value":{"type":"int","value":1001},"any-value-fails":false,` +
		`"nearest-passing":{"type":"int","value":1000}}`
)

// The members of a vector and of a spec that the paths of the tests name.
const (
	argsAt      = "args"
	casesAt     = "cases"
	choicesAt   = "choices"
	detailAt    = "detail"
	errorAt     = "error"
	expectAt    = "expect"
	generatorAt = "generator"
	genAt       = "gen"
	historyAt   = "history"
	kindAt      = "kind"
	ofAt        = "of"
	recordedAt  = "recorded"
	seedAt      = "seed"
	settingsAt  = "settings"
	shapeAt     = "shape"
	subjectAt   = "subject"
	typeAt      = "type"
	valueAt     = "value"
)

// TestHelpers checks the steps that the runners of every kind share: the
// vector's JSON parses, its seed is a decimal number, and its typed
// literals decode. Written with testing rather than with this library,
// because a verdict is not written with the subject.
func TestHelpers(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		for _, kind := range kinds {
			t.Run("returns a fault at the id for a "+string(kind)+" vector that is no JSON object", func(t *testing.T) {
				t.Parallel()
				err := check(t, kind, `[]`)
				expectFault(t, err, inVector(), notParsed)
				if _, ok := errors.AsType[*json.UnmarshalTypeError](err); !ok {
					t.Fatalf("Check returns %v, want one caused by the decoder", err)
				}
			})
		}

		tests := []struct {
			name       string
			kind       conformance.VectorKind
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name:       "returns a fault at the seed of a generation vector whose seed is no decimal number",
				kind:       conformance.Generation,
				give:       generated(digitGenerator, "forty-two", 0, ""),
				wantPath:   inVector(fault.Field(seedAt)),
				wantReason: noSeed,
			},
			{
				name:       "returns a fault at the seed of a shrinking vector whose seed is no decimal number",
				kind:       conformance.Shrinking,
				give:       shrunk(digitGenerator, `{"kind":"never"}`, "seven", passedOutputs),
				wantPath:   inVector(fault.Field(seedAt)),
				wantReason: noSeed,
			},
			{
				name:       "returns a fault at the seed of the settings of a behaviour vector whose seed is no number",
				kind:       conformance.Behaviour,
				give:       behaving(passingBody, `{"seed":"seven"}`, passedDetail),
				wantPath:   inVector(fault.Field(settingsAt), fault.Field(seedAt)),
				wantReason: noSeed,
			},
			{
				name:       "returns a fault at the seed of a forms vector whose seed is no decimal number",
				kind:       conformance.Forms,
				give:       `{"form":"prop-nil","subjects":["returns-null"],"shape":{"shape":"bool"},"seed":"x"}`,
				wantPath:   inVector(fault.Field(seedAt)),
				wantReason: noSeed,
			},
			{
				name:       "returns a fault at the type of a value that is no typed literal",
				kind:       conformance.Decoding,
				give:       decoded(digitGenerator, `[7]`, `[7]`, widget),
				wantPath:   inVector(fault.Field(valueAt), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at the expectation of a vector that expects neither pass nor fail",
				kind:       conformance.PathAbsent,
				give:       filesVector(aFile, pathArgs("b.txt"), "maybe"),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: `the vector expects "maybe", neither pass nor fail`,
			},
			{
				name:       "returns a fault at the expectation of a pass for a call that fails",
				kind:       conformance.PathAbsent,
				give:       filesVector(aFile, pathArgs("a.txt"), "pass"),
				wantPath:   inVector(fault.Field(expectAt)),
				wantReason: "the check ends as fail, want pass",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, tt.kind, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// noSeed is the reason of a seed that is no decimal number.
const noSeed = "the seed is no decimal number of 64 bits"

// modeMember matches a member of a tree literal that states a mode or an
// executable file.
var modeMember = regexp.MustCompile(`"mode"\s*:|"executable"\s*:\s*true`)

// mustRecordModes skips t on a platform whose file systems record no
// permission bits, where a tree reads no mode and no execute bit.
func mustRecordModes(t *testing.T) {
	t.Helper()
	if err := filetree.ModesUnrecorded(); err != nil {
		t.Skip(err)
	}
}

// mustStore skips t for a files vector that a platform whose file systems
// record no permission bits cannot store, as the definition's rules of
// conformance state: a vector of has-mode, and one that states a mode or an
// executable file. A files vector is one that states a workspace.
func mustStore(t *testing.T, v conformance.Vector) {
	t.Helper()
	var files struct {
		Workspace json.RawMessage `json:"workspace"`
	}
	_ = json.Unmarshal(v.Raw, &files)
	if files.Workspace != nil && (v.Kind == conformance.HasMode || modeMember.Match(v.Raw)) {
		mustRecordModes(t)
	}
}

// inVector returns the path of a fault at segs inside the vector or the
// case that a test builds.
func inVector(segs ...fault.Segment) fault.Path {
	return append(fault.Path{fault.Field(builtID)}, segs...)
}

// check returns what Check returns for a vector of kind whose case is the
// JSON text raw, with a directory of its own for stored cases.
func check(t *testing.T, kind conformance.VectorKind, raw string) error {
	t.Helper()
	return conformance.Vector{Kind: kind, ID: builtID, Raw: json.RawMessage(raw)}.Check(t.TempDir())
}

// expectFault fails t unless err is nil for an empty wantReason, or a fault
// at wantPath with wantReason.
func expectFault(t *testing.T, err error, wantPath fault.Path, wantReason string) {
	t.Helper()
	if wantReason == "" {
		if err != nil {
			t.Fatalf("returns %v, want nil", err)
		}
		return
	}
	f, ok := errors.AsType[*fault.Error](err)
	if !ok {
		t.Fatalf("returns %v, want a fault", err)
	}
	if !slices.Equal(f.Path, wantPath) || f.Reason != wantReason {
		t.Fatalf("returns the fault %q at %s, want %q at %s", f.Reason, f.Path, wantReason, wantPath)
	}
}

// invoking returns the script entry of an invocation of read without keys
// under the number n by client.
func invoking(n, client int) string {
	return fmt.Sprintf(`{"invoke":%d,"client":%d,"operation":"read","args":[],"keys":[]}`, n, client)
}

// decoded returns a decoding vector of generator that replays choices,
// records recorded, and decodes the typed literal value, or rejects the
// case for a value of null.
func decoded(generator, choices, recorded, value string) string {
	return fmt.Sprintf(`{"generator":%s,"choices":%s,"recorded":%s,"value":%s,"rejected":%t}`,
		generator, choices, recorded, value, value == null)
}

// behaving returns a behaviour vector of the run of body under settings,
// which states detail.
func behaving(body, settings, detail string) string {
	return fmt.Sprintf(`{"body":%s,"settings":%s,"detail":%s}`, body, settings, detail)
}

// The tree literals of the workspaces of the files vectors that the tests
// build.
const (
	// noFiles is the tree of no entries.
	noFiles = `{"type":"tree","entries":[]}`
	// aFile is the tree of the file a.txt, of the text a.
	aFile = `{"type":"tree","entries":[{"path":"a.txt","text":"a"}]}`
)

// filesVector returns a files vector of the workspace and the arguments,
// each a JSON text, that expects expect.
func filesVector(workspace, args, expect string) string {
	return fmt.Sprintf(`{"workspace":%s,"args":%s,"expect":%q}`, workspace, args, expect)
}

// pathArgs returns the arguments of a files vector: the typed literal of the
// path, and after it the typed literals after.
func pathArgs(path string, after ...string) string {
	return "[" + strings.Join(append([]string{stringOf(path)}, after...), ",") + "]"
}

// stringOf returns the typed literal of the string s.
func stringOf(s string) string {
	return fmt.Sprintf(`{"type":"string","value":%q}`, s)
}

// failedDetail returns the detail of the run of bigBody under seven, of the
// minimal counterexample's one draw, the failure's identity, and the other
// failures, each a JSON text.
func failedDetail(draw, failure, others string) string {
	return fmt.Sprintf(`{"outcome":"counterexample","cases":1,"rejected":0,"seed":"7","counterexample":[%s],`+
		`"failure":%q,"choices":"prop1:AOkH","others":%s,"divergence":null,"coverage":null}`, draw, failure, others)
}
