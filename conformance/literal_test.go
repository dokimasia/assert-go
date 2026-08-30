// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/json"
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/conformance"
)

func TestLiteral(t *testing.T) {
	t.Parallel()

	t.Run("Decode", func(t *testing.T) {
		t.Parallel()

		scalars := []struct {
			name string
			in   string
			want any
		}{
			{"null", `{"type":"null"}`, nil},
			{"bool", `{"type":"bool","value":true}`, true},
			{"int", `{"type":"int","value":1}`, 1},
			{"float", `{"type":"float","value":1.5}`, 1.5},
			{"string", `{"type":"string","value":"abc"}`, "abc"},
		}
		for _, tc := range scalars {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got, err := conformance.Decode(json.RawMessage(tc.in))
				assert.NoError(t, err, "the literal decodes")
				assert.Equal(t, got, tc.want, "the literal decodes to its native value")
			})
		}

		t.Run("an empty list is present, not absent", func(t *testing.T) {
			t.Parallel()

			got, err := conformance.Decode(json.RawMessage(`{"type":"list","of":"int","value":[]}`))
			assert.NoError(t, err, "the literal decodes")
			assert.NotNil(t, got, "an empty list is not a nil slice")
			assert.Empty(t, got, "an empty list holds nothing")
		})

		t.Run("an empty map is present, not absent", func(t *testing.T) {
			t.Parallel()

			got, err := conformance.Decode(json.RawMessage(
				`{"type":"map","key":"string","of":"int","value":{}}`,
			))
			assert.NoError(t, err, "the literal decodes")
			assert.NotNil(t, got, "an empty map is not a nil map")
		})

		t.Run("NaN", func(t *testing.T) {
			t.Parallel()

			got, err := conformance.Decode(json.RawMessage(`{"type":"float","value":"NaN"}`))
			assert.NoError(t, err, "the literal decodes")
			assert.True(t, math.IsNaN(got.(float64)), "the literal decodes to NaN")
		})

		t.Run("infinity", func(t *testing.T) {
			t.Parallel()

			got, err := conformance.Decode(json.RawMessage(`{"type":"float","value":"Inf"}`))
			assert.NoError(t, err, "the literal decodes")
			assert.True(t, math.IsInf(got.(float64), 1), "the literal decodes to positive infinity")
		})

		t.Run("a populated map", func(t *testing.T) {
			t.Parallel()

			got, err := conformance.Decode(json.RawMessage(
				`{"type":"map","key":"string","of":"int","value":{"a":1}}`,
			))
			assert.NoError(t, err, "the literal decodes")

			// Asserted at its concrete type rather than through any:
			// the element type is part of what the literal states, and
			// a comparison through any would not check it.
			decoded, ok := got.(map[string]int)
			assert.True(t, ok, "the literal decodes to a string-keyed map of ints")
			assert.Equal(t, decoded, map[string]int{"a": 1}, "the map holds what the literal states")
		})

		refused := []struct{ name, in string }{
			{"an unknown type", `{"type":"widget"}`},
			{"a list of an unknown element type", `{"type":"list","of":"widget","value":[]}`},
			{"a map keyed by something other than a string", `{"type":"map","key":"int","of":"int","value":{}}`},
			{"an unrecognised float name", `{"type":"float","value":"Huge"}`},
		}
		for _, tc := range refused {
			t.Run(tc.name+" is refused", func(t *testing.T) {
				t.Parallel()

				_, err := conformance.Decode(json.RawMessage(tc.in))
				assert.HasError(t, err, "the literal is refused rather than guessed at")
			})
		}
	})
}
