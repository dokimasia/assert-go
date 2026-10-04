// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// The shapes of the fixture types flag and tree, as the definition's
// fixtures vectors state them, and as the runner states them in a fault:
// with each definition that a ref names renamed in the order of the refs,
// and the keys of each object in order.
const (
	// flagShape is the shape of flag.
	flagShape = `{"shape":"record","fields":[["enabled",{"shape":"bool"}]]}`
	// flagText is flagShape in a fault.
	flagText = `{"fields":[["enabled",{"shape":"bool"}]],"shape":"record"}`
	// treeShape is the shape of tree, with a definition leaf that no ref
	// names.
	treeShape = `{"shape":"ref","name":"tree","definitions":{` +
		`"tree":{"shape":"record","fields":[["value",{"shape":"int","width":32,"signed":true}],` +
		`["children",{"shape":"list","of":{"shape":"ref","name":"tree"}}]]},` +
		`"leaf":{"shape":"bool"}}}`
	// treeDefinition is the definition of tree in a fault.
	treeDefinition = `{"fields":[["value",{"shape":"int","signed":true,"width":32}],` +
		`["children",{"of":{"name":"#0","shape":"ref"},"shape":"list"}]],"shape":"record"}`
)

// TestFixture checks a fixtures vector: the shape that a fixture type of
// the definition reads as. Written with testing rather than with this
// library, because a verdict is not written with the subject.
func TestFixture(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			give       string
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns nil for the shape of a fixture type",
				give: `{"fixture":"flag","shape":` + flagShape + `}`,
			},
			{
				name:       "returns a fault at the fixture for a fixtures vector that names no fixture type",
				give:       `{"fixture":"widget","shape":{"shape":"bool"}}`,
				wantPath:   inVector(fault.Field("fixture")),
				wantReason: `"widget" is no fixture type`,
			},
			{
				name: "returns a fault at the shape for a shape other than the fixture type's",
				give: `{"fixture":"flag","shape":{"shape":"record","fields":[["enabled",` +
					`{"shape":"int","width":32,"signed":true,"min":-1000,"max":1000}]]}}`,
				wantPath: inVector(fault.Field(shapeAt)),
				wantReason: "the fixture type reads as " + flagText + `, want {"fields":[["enabled",` +
					`{"max":1000,"min":-1000,"shape":"int","signed":true,"width":32}]],"shape":"record"}`,
			},
			{
				name:     "returns a fault at the shape for a definition that no ref of the shape names",
				give:     `{"fixture":"tree","shape":` + treeShape + `}`,
				wantPath: inVector(fault.Field(shapeAt)),
				wantReason: `the fixture type reads as {"definitions":{"#0":` + treeDefinition +
					`},"name":"#0","shape":"ref"}, want {"definitions":{"#0":` + treeDefinition +
					`,"leaf":{"shape":"bool"}},"name":"#0","shape":"ref"}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Fixtures, tt.give), tt.wantPath, tt.wantReason)
			})
		}
	})
}
