// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	bignum "math/big"
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/prop"
)

// TestTag checks how the reader parses a field's prop tag: the keys that it
// takes, the form of each value in the shape file, and each fault of a tag
// at its field.
func TestTag(t *testing.T) {
	t.Parallel()

	t.Run("ShapeOf", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give func() (string, error)
			want string
		}{
			{
				name: "states an integer bound of 2^53 - 1 as a number",
				give: prop.ShapeOf[struct {
					V int64 `prop:"max=9007199254740991"`
				}],
				want: `{"shape":"int","width":64,"signed":true,"max":9007199254740991}`,
			},
			{
				name: "states an integer bound beyond 2^53 - 1 as the decimal string of a typed literal",
				give: prop.ShapeOf[struct {
					V int64 `prop:"min=-9007199254740993"`
				}],
				want: `{"shape":"int","width":64,"signed":true,"min":"-9007199254740993"}`,
			},
			{
				name: "states a bound of a decimal as a string",
				give: prop.ShapeOf[struct {
					V *bignum.Rat `prop:"scale=1,max=2.5"`
				}],
				want: `{"shape":"decimal","scale":1,"max":"2.5"}`,
			},
			{
				name: "states a bound of a date as a string",
				give: prop.ShapeOf[struct {
					V time.Time `prop:"date,min=2020-01-01"`
				}],
				want: `{"shape":"date","min":"2020-01-01"}`,
			},
			{
				name: "takes the rest of the tag after pattern= as the pattern, commas included",
				give: prop.ShapeOf[struct {
					V string `prop:"pattern=[a-z]{2,3}"`
				}],
				want: `{"shape":"string","pattern":"[a-z]{2,3}"}`,
			},
			{
				name: "takes the rest of the tag after alphabet= as the alphabet",
				give: prop.ShapeOf[struct {
					V rune `prop:"char,alphabet=a,b"`
				}],
				want: `{"shape":"char","alphabet":"a,b"}`,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, shapeTree(t, tt.give), jsonTree(t, `{"shape":"record","fields":[["V",`+tt.want+`]]}`),
					"the shape of the field")
			})
		}

		faults := []struct {
			name       string
			give       func() (string, error)
			wantPath   fault.Path
			wantReason string
		}{
			{
				name: "returns a fault at a field whose tag states a key that the prop tag does not have",
				give: prop.ShapeOf[struct {
					V int8 `prop:"mni=1"`
				}],
				wantPath:   fieldV("int8", "mni=1"),
				wantReason: `the key "mni" is no key of the prop tag`,
			},
			{
				name: "returns a fault at a field whose tag states a value of a flag",
				give: prop.ShapeOf[struct {
					V rune `prop:"char=1"`
				}],
				wantPath:   fieldV("int32", "char=1"),
				wantReason: "the key char takes no value",
			},
			{
				name: "returns a fault at a field whose tag states a key without its value",
				give: prop.ShapeOf[struct {
					V int8 `prop:"min"`
				}],
				wantPath:   fieldV("int8", "min"),
				wantReason: "the key min states no value",
			},
			{
				name: "returns a fault at a field whose tag states a key twice",
				give: prop.ShapeOf[struct {
					V int8 `prop:"min=1,min=2"`
				}],
				wantPath:   fieldV("int8", "min=1,min=2"),
				wantReason: "the tag states the key min twice",
			},
			{
				name: "returns a fault at a field whose tag ends in a comma",
				give: prop.ShapeOf[struct {
					V int8 `prop:"min=1,"`
				}],
				wantPath:   fieldV("int8", "min=1,"),
				wantReason: `the tag "min=1," ends in a comma`,
			},
			{
				name: "returns a fault at a time.Time whose tag chooses two shapes",
				give: prop.ShapeOf[struct {
					V time.Time `prop:"date,local-date-time"`
				}],
				wantPath:   fieldV("time.Time", "date,local-date-time"),
				wantReason: "the tag states date and local-date-time, which choose two shapes",
			},
			{
				name: "returns a fault at an integer whose tag chooses two shapes",
				give: prop.ShapeOf[struct {
					V int32 `prop:"offset,char"`
				}],
				wantPath:   fieldV("int32", "offset,char"),
				wantReason: "the tag states offset and char, which choose two shapes",
			},
		}
		for _, tt := range faults {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := tt.give()
				expectFault(t, err, fault.Error{Op: shapeOfOp, Path: tt.wantPath, Reason: tt.wantReason})
			})
		}
	})
}
