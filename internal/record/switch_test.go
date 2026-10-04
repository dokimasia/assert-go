// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package record_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/internal/childtest"
	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/record"
)

// TestSwitch checks On in a child process for each value of
// DOKIMI_ASSERT_RECORD, because a process reads the variable once. The
// checks in the child use the package testing alone: under a switch of
// another value, every assertion of this module ends with the switch's
// fault.
func TestSwitch(t *testing.T) {
	t.Parallel()

	t.Run("On", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name      string
			giveEnv   []string
			want      bool
			wantFault bool
		}{
			{name: "returns false for an unset variable"},
			{name: "returns false for an empty value", giveEnv: []string{record.Variable + "="}},
			{name: "returns false for 0", giveEnv: []string{record.Variable + "=0"}},
			{name: "returns true for 1", giveEnv: []string{record.Variable + "=1"}, want: true},
			{
				name:      "returns false and ErrSwitch at the variable for any other value",
				giveEnv:   []string{record.Variable + "=yes"},
				wantFault: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				if !childtest.InChild(t) {
					runChild(t, tt.giveEnv...)
					return
				}
				on, err := record.On()
				if again, _ := record.On(); on != tt.want || again != on {
					t.Fatalf("On() = %v, then %v, want %v twice", on, again, tt.want)
				}
				if !tt.wantFault {
					if err != nil {
						t.Fatalf("On() = %v, want no fault", err)
					}
					return
				}
				var f *fault.Error
				if !errors.As(err, &f) || f.Op != "record.On" || !errors.Is(err, record.ErrSwitch) ||
					len(f.Path) != 1 || f.Path[0] != fault.Field(record.Variable) {
					t.Fatalf("On() = %#v, want a fault of record.On of the kind ErrSwitch at the variable", err)
				}
			})
		}
	})
}

// TestSwitchAllocs checks that On allocates nothing after its first call.
func TestSwitchAllocs(t *testing.T) {
	_, _ = record.On()
	assert.MaxAllocs(t, func() { _, _ = record.On() }, 0, "On allocates nothing")
}

// BenchmarkSwitch measures On after its first call.
func BenchmarkSwitch(b *testing.B) {
	b.Run("On", func(b *testing.B) {
		_, _ = record.On()
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			_, _ = record.On()
		}
	})
}
