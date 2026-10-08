// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"strconv"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The shape and the details of the forms vectors that the tests build.
const (
	// thousands is the shape of the integers from -1000 to 1000, which
	// OfShape decodes to an int32.
	thousands = `{"shape":"int","width":32,"signed":true,"min":-1000,"max":1000}`
	// passed is the detail of a passing run of 100 cases of the seed 7.
	passed = `{"outcome":"passed","cases":100,"rejected":0,"seed":"7","counterexample":null,"failure":null,` +
		`"choices":null,"others":null,"divergence":null,"coverage":null}`
)

// TestForms checks a forms vector: a property form run on built subjects
// over the generator of a shape. Written with testing rather than with
// this library, because a verdict is not written with the subject.
func TestForms(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		atFailure := func(segs ...fault.Segment) fault.Path {
			return inVector(append([]fault.Segment{
				fault.Field(detailAt), fault.Field("failure"), fault.Field(detailAt),
			}, segs...)...)
		}
		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for a subject that reads an absent handle as success",
				give: forms("prop-nil-context-safe", `["reads-handle"]`, `[]`, thousands, passed),
			},
			{
				name: "returns nil for the failing run that the vector states",
				give: forms(
					"prop-nil",
					`["identity"]`,
					`[]`,
					thousands,
					nilFails(0, `{"got":{"type":"int","value":0}}`),
				),
			},
			{
				name:       "returns a fault at the form for a form that no vector runs",
				give:       forms("prop-max-allocs", `["returns-ok"]`, `[]`, thousands, passed),
				wantPath:   inVector(fault.Field("form")),
				wantReason: `"prop-max-allocs" is no form that a vector runs`,
			},
			{
				name:       "returns a fault at the shape for a shape that does not read",
				give:       forms("prop-nil", `["returns-null"]`, `[]`, `{"shape":"widget"}`, passed),
				wantPath:   inVector(fault.Field(shapeAt)),
				wantReason: "the shape does not read",
			},
			{
				name: "returns a fault at the element shape for an element shape that does not read alone",
				give: forms("prop-nil", `["returns-null"]`, `[]`,
					`{"shape":"list","of":{"shape":"ref","name":"leaf"},"definitions":{"leaf":{"shape":"bool"}}}`,
					passed),
				wantPath:   inVector(fault.Field(shapeAt), fault.Field(ofAt)),
				wantReason: "the shape of the elements does not read alone",
			},
			{
				name:       "returns a fault at the subjects for another number of subjects than the form takes",
				give:       forms("prop-nil", `[]`, `[]`, thousands, passed),
				wantPath:   inVector(fault.Field("subjects")),
				wantReason: "the form takes 1 subjects, and the vector states 0",
			},
			{
				name:       "returns a fault at the subject for a subject that the form does not take",
				give:       forms("prop-nil", `["adds"]`, `[]`, thousands, passed),
				wantPath:   inVector(fault.Field("subjects"), fault.Index(0)),
				wantReason: `"adds" is no subject that the form takes`,
			},
			{
				name:       "returns a fault at the subject for a subject that the definition does not state",
				give:       forms("prop-nil", `["sleeps"]`, `[]`, thousands, passed),
				wantPath:   inVector(fault.Field("subjects"), fault.Index(0)),
				wantReason: `"sleeps" is no subject that the form takes`,
			},
			{
				name:       "returns a fault at the value for a value that is no typed literal",
				give:       forms("prop-length", `["identity"]`, `[`+widget+`]`, thousands, passed),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(0), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name:       "returns a fault at the values for another number of values than the form takes",
				give:       forms("prop-nil", `["identity"]`, `[`+four+`]`, thousands, passed),
				wantPath:   inVector(fault.Field(argsAt)),
				wantReason: "the form takes 0 values, and the vector states 1",
			},
			{
				name:       "returns a fault at the value for a value of another type than the form takes",
				give:       forms("prop-length", `["identity"]`, `[{"type":"string","value":"3"}]`, thousands, passed),
				wantPath:   inVector(fault.Field(argsAt), fault.Index(0)),
				wantReason: "the value is a string, and the form takes a int",
			},
			{
				name:       "returns a fault at the outcome for a counterexample of a run that passes",
				give:       forms("prop-nil", `["returns-null"]`, `[]`, thousands, nilFails(0, `{}`)),
				wantPath:   inVector(fault.Field(detailAt), fault.Field("outcome")),
				wantReason: `the run states "passed", want "counterexample"`,
			},
			{
				name: "returns a fault at the cases for counts other than the passing run's",
				give: forms("prop-nil", `["returns-null"]`, `[]`, thousands,
					`{"outcome":"passed","cases":99,"rejected":0,"seed":"7"}`),
				wantPath:   inVector(fault.Field(detailAt), fault.Field(casesAt)),
				wantReason: "the run states 100, want 99",
			},
			{
				name:       "returns a fault at the cases for a detail other than the failing run's",
				give:       forms("prop-nil", `["identity"]`, `[]`, thousands, nilFails(5, `{}`)),
				wantPath:   inVector(fault.Field(detailAt), fault.Field(casesAt)),
				wantReason: "the run states 0, want 5",
			},
			{
				name: "returns a fault at the field of the failure that its record does not state",
				give: forms(
					"prop-nil",
					`["identity"]`,
					`[]`,
					thousands,
					nilFails(0, `{"widget":{"type":"null"}}`),
				),
				wantPath:   atFailure(fault.Key("widget")),
				wantReason: `the failure states no such field, want {"type":"null"}`,
			},
			{
				name:       "returns a fault at the field of the failure that is no typed literal",
				give:       forms("prop-nil", `["identity"]`, `[]`, thousands, nilFails(0, `{"got":`+widget+`}`)),
				wantPath:   atFailure(fault.Key("got"), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
			{
				name: "returns a fault at the field of the failure other than its record's",
				give: forms("prop-nil", `["identity"]`, `[]`, thousands,
					nilFails(0, `{"got":{"type":"int","value":1}}`)),
				wantPath:   atFailure(fault.Key("got")),
				wantReason: `the field is int:0, want {"type":"int","value":1}`,
			},
			{
				name: "returns nil for a run over a generator that states no shape",
				give: `{"form":"prop-nil-context-safe","subjects":["reads-handle"],"args":[],` +
					`"generator":{"gen":"integer","min":-1000,"max":1000},"seed":"7","detail":` + passed + `}`,
			},
			{
				name: "returns a fault for a vector that states a shape and a generator",
				give: `{"form":"prop-nil","subjects":["returns-null"],"args":[],"shape":` + thousands +
					`,"generator":{"gen":"integer","min":0,"max":9},"seed":"7","detail":` + passed + `}`,
				wantPath:   inVector(),
				wantReason: "the vector states one of a shape and a generator",
			},
			{
				name: "returns a fault at the generator for a generator that does not read",
				give: `{"form":"prop-nil","subjects":["returns-null"],"args":[],"generator":{"gen":"widget"},` +
					`"seed":"7","detail":` + passed + `}`,
				wantPath:   inVector(fault.Field("generator"), fault.Field(genAt)),
				wantReason: `"widget" names no generator`,
			},
			{
				name: "returns a fault at the example for an example that is no typed literal",
				give: `{"form":"prop-nil","subjects":["returns-null"],"args":[],"shape":` + thousands +
					`,"examples":[` + widget + `],"seed":"7","detail":` + passed + `}`,
				wantPath:   inVector(fault.Field("examples"), fault.Index(0), fault.Field(typeAt)),
				wantReason: unknownWidget,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Forms, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}

// forms returns a forms vector of form, over the subjects and the args, whose
// input has the shape, of the seed 7, which states detail.
func forms(form, subjects, args, shape, detail string) string {
	return `{"form":"` + form + `","subjects":` + subjects + `,"args":` + args + `,"shape":` + shape +
		`,"seed":"7","detail":` + detail + `}`
}

// nilFails returns the detail of the run of nil over identity on the
// integers from -1000 to 1000 of the seed 7, which fails at its first case
// of 0, with the cases and the failure's detail that the detail states.
func nilFails(cases int, failure string) string {
	return `{"outcome":"counterexample","cases":` + strconv.Itoa(cases) + `,"rejected":0,"seed":"7",` +
		`"counterexample":[{"label":"input","value":{"type":"int","value":0},"any-value-fails":true,` +
		`"nearest-passing":null}],"failure":{"assertion":"nil","detail":` + failure + `},` +
		`"choices":"prop1:AAA","others":[],"divergence":null,"coverage":null}`
}
