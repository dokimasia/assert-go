// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"testing"
	"time"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
	"go.dokimi.dev/assert/prop"
)

// optionalPayload is an enum of a variant without a payload and a variant
// whose payload is an optional.
const optionalPayload = `{"shape":"enum","variants":[["none",null],` +
	`["some",{"shape":"optional","of":{"shape":"bool"}}]]}`

// TestValue checks the Go values of an enum and a wall time: a Variant
// keeps a variant without a payload apart from one whose payload is an
// absent optional, and a WallTime states a wall time and its zone.
func TestValue(t *testing.T) {
	t.Parallel()

	t.Run("OfShape", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give []choice.Choice
			want prop.Variant
		}{
			{
				name: "decodes a variant without a payload to a Variant without one",
				give: []choice.Choice{integer(0)},
				want: prop.Variant{Name: "none"},
			},
			{
				name: "decodes a variant whose payload is an absent optional to a Variant with a nil payload",
				give: []choice.Choice{integer(1), integer(0)},
				want: prop.Variant{Name: "some", HasPayload: true},
			},
			{
				name: "decodes a variant with a payload to a Variant of the payload's Go value",
				give: []choice.Choice{integer(1), integer(1), integer(1)},
				want: prop.Variant{Name: "some", Payload: true, HasPayload: true},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				g, err := prop.OfShape(optionalPayload)
				assert.NoError(t, err, "the shape reads")
				got := decodedBy(g, tt.give...)
				assert.Equal(t, got, any(tt.want), "the variant")
				choices, err := engine.Invert(engine.Generator[any](g), got)
				assert.NoError(t, err, "the Variant runs back to its choices")
				assert.Equal(t, choices, tt.give, "the choices of the variant")
			})
		}

		t.Run("decodes a wall time to a WallTime of its local date and time and its zone", func(t *testing.T) {
			t.Parallel()
			g, err := prop.OfShape(`{"shape":"wall-time","unit":"s"}`)
			assert.NoError(t, err, "the shape reads")
			want := prop.WallTime{Local: time.Date(1970, time.January, 2, 0, 1, 0, 0, time.UTC), Zone: time.UTC}
			assert.Equal(t, decodedBy(g, integer(0), signed(1), signed(60)), any(want), "the wall time in UTC")
		})
	})
}
