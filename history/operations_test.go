// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package history_test

import (
	"math"
	"reflect"
	"testing"

	"go.dokimi.dev/assert"
)

// count is a named type over int, whose states the defaults compare through
// reflection.
type count int

// reading is a struct state of one float, whose states the defaults compare
// through reflection.
type reading struct {
	// Value is the reading.
	Value float64
}

// TestOperations checks how a check compares and hashes the states of a
// model: the defaults for a basic type and for any other type, and an Equal
// and a Hash that the model states.
func TestOperations(t *testing.T) {
	t.Parallel()

	negativeZero := math.Copysign(0, -1)

	t.Run("Linearizable", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give int
			want int
		}{
			{name: "takes two equal bools for one state", give: left(true, true), want: 1},
			{name: "takes two different bools for two states", give: left(true, false), want: 2},
			{name: "takes two equal ints for one state", give: left(7, 7), want: 1},
			{name: "takes two different ints for two states", give: left(7, 8), want: 2},
			{name: "takes two equal int8s for one state", give: left[int8](7, 7), want: 1},
			{name: "takes two different int8s for two states", give: left[int8](7, 8), want: 2},
			{name: "takes two equal int16s for one state", give: left[int16](7, 7), want: 1},
			{name: "takes two different int16s for two states", give: left[int16](7, 8), want: 2},
			{name: "takes two equal int32s for one state", give: left[int32](7, 7), want: 1},
			{name: "takes two different int32s for two states", give: left[int32](7, 8), want: 2},
			{name: "takes two equal int64s for one state", give: left[int64](7, 7), want: 1},
			{name: "takes two different int64s for two states", give: left[int64](7, 8), want: 2},
			{name: "takes two equal uints for one state", give: left[uint](7, 7), want: 1},
			{name: "takes two different uints for two states", give: left[uint](7, 8), want: 2},
			{name: "takes two equal uint8s for one state", give: left[uint8](7, 7), want: 1},
			{name: "takes two different uint8s for two states", give: left[uint8](7, 8), want: 2},
			{name: "takes two equal uint16s for one state", give: left[uint16](7, 7), want: 1},
			{name: "takes two different uint16s for two states", give: left[uint16](7, 8), want: 2},
			{name: "takes two equal uint32s for one state", give: left[uint32](7, 7), want: 1},
			{name: "takes two different uint32s for two states", give: left[uint32](7, 8), want: 2},
			{name: "takes two equal uint64s for one state", give: left[uint64](7, 7), want: 1},
			{name: "takes two different uint64s for two states", give: left[uint64](7, 8), want: 2},
			{name: "takes two equal uintptrs for one state", give: left[uintptr](7, 7), want: 1},
			{name: "takes two different uintptrs for two states", give: left[uintptr](7, 8), want: 2},
			{name: "takes a float32 -0 and +0 for one state", give: left(float32(negativeZero), 0), want: 1},
			{
				name: "takes two float32 NaNs for two states",
				give: left(float32(math.NaN()), float32(math.NaN())),
				want: 2,
			},
			{name: "takes a float64 -0 and +0 for one state", give: left(negativeZero, 0), want: 1},
			{name: "takes two float64 NaNs for two states", give: left(math.NaN(), math.NaN()), want: 2},
			{name: "takes two equal strings for one state", give: left("a", "a"), want: 1},
			{name: "takes two different strings for two states", give: left("a", "b"), want: 2},
			{name: "takes two equal states of a named type for one state", give: left[count](7, 7), want: 1},
			{name: "takes two different states of a named type for two states", give: left[count](7, 8), want: 2},
			{
				name: "takes struct states of -0 and +0 for one state",
				give: left(reading{negativeZero}, reading{0}),
				want: 1,
			},
			{
				name: "takes struct states of NaNs for two states",
				give: left(reading{math.NaN()}, reading{math.NaN()}),
				want: 2,
			},
			{name: "takes a nil and an empty slice for two states", give: left([]int(nil), []int{}), want: 2},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.give, tt.want, "the states of the frontier after the write")
			})
		}

		t.Run("compares states with the Equal that the model states", func(t *testing.T) {
			t.Parallel()
			m := spreading(0, 1, 3)
			m.Equal = sameParity
			got := detailOf(violatedRead(), m)
			assert.Equal(t, got[statesField], any([]int{1}), "3 has the parity of 1, which comes first")
		})

		t.Run("hashes every state alike for a model that states Equal alone", func(t *testing.T) {
			t.Parallel()
			m := register
			m.Equal = sameParity
			got := detailOf(parityWrites(), m)
			assert.Equal(t, got[stepsField], any(5), "the memo finds the configuration of 1 then 3 after 3 then 1")
		})

		t.Run("hashes states with the Hash that the model states beside the default comparison", func(t *testing.T) {
			t.Parallel()
			hashed := 0
			m := register
			m.Hash = func(s int) uint64 {
				hashed++
				return uint64(s)
			}
			assert.Equal(t, detailOf(violatedRead(), m), detailOf(violatedRead(), register), "the record of the model")
			assert.Equal(t, hashed, 1, "the one state that the write leaves")
		})

		t.Run("takes two states of different hashes for two states, whatever Equal reports", func(t *testing.T) {
			t.Parallel()
			m := spreading(0, 1, 2)
			m.Equal = func(int, int) bool { return true }
			m.Hash = func(s int) uint64 { return uint64(s) }
			got := detailOf(violatedRead(), m)
			assert.Equal(t, got[statesField], any([]int{1, 2}), "the hash tells 1 and 2 apart")
		})
	})
}

// left checks the violated read against the model whose write leaves states,
// and returns the number of states of the frontier, which the write leaves.
func left[S any](states ...S) int {
	got := detailOf(violatedRead(), spreading(*new(S), states...))
	return reflect.ValueOf(got[statesField]).Len()
}

// sameParity reports whether a and b are both even or both odd.
func sameParity(a, b int) bool {
	return a%2 == b%2
}
