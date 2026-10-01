// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package engine_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert/internal/prop/choice"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// TestFloats checks the pass that rounds a fraction to fewer bits and
// lowers an integral float as an integer, through runs from stored failing
// cases, pinned to what the definition's executable reference reports.
func TestFloats(t *testing.T) {
	t.Parallel()

	unit := engine.Float(0.0, 10.0, choice.ExcludeNaN)
	signed := engine.Float(-10.0, 10.0, choice.ExcludeNaN)
	wide32 := engine.Float[float32](0, 1<<25, choice.ExcludeNaN)
	reals := engine.Float(math.Inf(-1), math.Inf(1), choice.ExcludeNaN)
	nans := engine.Float(math.Inf(-1), math.Inf(1), choice.AdmitNaN)
	bigger := func(c *engine.Case) string {
		return failsWhen(engine.Draw(c, unit, "v") > 2.5, "bigger")
	}
	huge := engine.Float(0.0, 0x1p60, choice.ExcludeNaN)
	thousand := func(c *engine.Case) string {
		return failsWhen(engine.Draw(c, huge, "v") >= 1000, "thousand")
	}

	t.Run("Run", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name   string
			p      property
			stored []choice.Choice
			want   reference
		}{
			{
				name: "rounds a fraction that must stay one to two fractional bits",
				p: func(c *engine.Case) string {
					value := engine.Draw(c, unit, "v")
					return failsWhen(value > 2.5 && value != math.Trunc(value), "fraction")
				},
				stored: []choice.Choice{float(2.875)},
				want: reference{
					explanation: []engine.Explained{{Label: "v", Value: 2.75, Relevance: engine.ValueMatters}},
					token:       "prop1:AgAAAAAAAAZA",
					runs:        7,
					calls:       9,
					digest:      "97970c7c1a808c356a401342dc4567402d9dcdde24712a9823d5172891c70644",
				},
			},
			{
				name:   "rounds a fraction away from its target to an integer",
				p:      bigger,
				stored: []choice.Choice{float(2.875)},
				want: reference{
					explanation: []engine.Explained{{Label: "v", Value: 3.0, Relevance: engine.ValueMatters}},
					token:       "prop1:AgAAAAAAAAhA",
					runs:        9,
					calls:       11,
					digest:      "8350016ab6a624b4ac05b75018c9316664b3186653bb76769220f709d86b2761",
				},
			},
			{
				name:   "lowers an integral float as an integer",
				p:      bigger,
				stored: []choice.Choice{float(9)},
				want: reference{
					explanation: []engine.Explained{{Label: "v", Value: 3.0, Relevance: engine.ValueMatters}},
					token:       "prop1:AgAAAAAAAAhA",
					runs:        10,
					calls:       12,
					digest:      "8b9c1062e7f3a8cc89942e818d3b73f522304bb8895e9b36e639b3b84ba78e70",
				},
			},
			{
				name:   "lowers an integral float of magnitude 2^53 - 1 as an integer",
				p:      thousand,
				stored: []choice.Choice{float(1<<53 - 1)},
				want: reference{
					explanation: []engine.Explained{{Label: "v", Value: 1000.0, Relevance: engine.ValueMatters}},
					token:       "prop1:AgAAAAAAQI9A",
					runs:        65,
					calls:       67,
					digest:      "16e9edf1eefbc1b7046eb732ae2020f004039af132bd215765eb097fbebc9a91",
				},
			},
			{
				name:   "leaves an integral float of magnitude 2^53, which it cannot lower as an integer",
				p:      thousand,
				stored: []choice.Choice{float(1 << 53)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "v", Value: 9007199254740992.0, Relevance: engine.ValueMatters},
					},
					token:  "prop1:AgAAAAAAAEBD",
					runs:   3,
					calls:  5,
					digest: "daab08421559bdab799cf7daf681bc24f50a5269bc97218919e3dbe980b7348c",
				},
			},
			{
				name: "moves a negative integral float to the simpler value above zero",
				p: func(c *engine.Case) string {
					return failsWhen(math.Abs(engine.Draw(c, signed, "v")) >= 3, "large")
				},
				stored: []choice.Choice{float(-9)},
				want: reference{
					explanation: []engine.Explained{{Label: "v", Value: 3.0, Relevance: engine.ValueMatters}},
					token:       "prop1:AgAAAAAAAAhA",
					runs:        12,
					calls:       14,
					digest:      "636916172826c617622aeceb7749c264d171aa709d8a14618fcc431fe117e70b",
				},
			},
			{
				name: "skips the integers that a float32 cannot state",
				p: func(c *engine.Case) string {
					return failsWhen(engine.Draw(c, wide32, "v") >= 1<<24+2, "big")
				},
				stored: []choice.Choice{float(1 << 25)},
				want: reference{
					explanation: []engine.Explained{
						{Label: "v", Value: float32(1<<24 + 2), Relevance: engine.ValueMatters},
					},
					token:  "prop1:AgAAACAAAHBB",
					runs:   50,
					calls:  52,
					digest: "cbd5932845723b136343fe52b18286e23b2d7291234be7570300b531d8fb332a",
				},
			},
			{
				name: "leaves an infinity, which has no fraction to round",
				p: func(c *engine.Case) string {
					return failsWhen(math.IsInf(engine.Draw(c, reals, "v"), 0), "infinite")
				},
				stored: []choice.Choice{float(math.Inf(1))},
				want: reference{
					explanation: []engine.Explained{{Label: "v", Value: math.Inf(1), Relevance: engine.ValueMatters}},
					token:       "prop1:AgAAAAAAAPB_",
					runs:        3,
					calls:       5,
					digest:      "53653c13f12d67f2573a488e80d41bf6012756c1d504d7bd22c492e8a0883130",
				},
			},
			{
				name: "leaves NaN, which has no fraction to round",
				p: func(c *engine.Case) string {
					return failsWhen(math.IsNaN(engine.Draw(c, nans, "v")), "nan")
				},
				stored: []choice.Choice{float(math.NaN())},
				want: reference{
					explanation: []engine.Explained{{Label: "v", Value: math.NaN(), Relevance: engine.ValueMatters}},
					token:       "prop1:AgAAAAAAAPh_",
					runs:        3,
					calls:       5,
					digest:      "87060ab8a352446b0c5b0aa74de99a2790b409b3157526f4d85d7bf538831032",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, trace := traced(tt.p, settled(tt.stored...))
				matchesReference(t, got, trace, tt.want)
			})
		}
	})
}
