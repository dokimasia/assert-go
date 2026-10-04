// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package prop_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

// The parts of a run's sentence, for runs of seed 7.
const (
	// header is the first line of a run that ended with an outcome after a
	// number of valid cases and none rejected.
	header = "the property holds: %s after %d valid and 0 rejected cases, seed 7"
	// replayZero is the replay line of the case of one integer choice of 0.
	replayZero = "\nreplay: prop.Replay(\"prop1:AAA\") or DOKIMI_ASSERT_PROP_REPLAY=prop1:AAA"
)

// TestRender checks the sentence of a failing run's record that a seat
// receives when it takes no record.
func TestRender(t *testing.T) {
	t.Parallel()

	t.Run("ForAll", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			body func(*prop.Case)
			opts []prop.Option
			want string
		}{
			{
				name: "states the counterexample with the nearest passing value and the replay",
				body: failsAtLeast(10000, 1001, big),
				want: fmt.Sprintf(header, "counterexample", 1) +
					"\n  value: 1001, 1000 passes" +
					"\nfailure of big: " +
					"\nreplay: prop.Replay(\"prop1:AOkH\") or DOKIMI_ASSERT_PROP_REPLAY=prop1:AOkH",
			},
			{
				name: "states a draw where any value fails",
				body: failsAtLeast(100, 0, every),
				want: fmt.Sprintf(header, "counterexample", 0) +
					"\n  value: 0, any value fails" +
					"\nfailure of every: " +
					replayZero,
			},
			{
				name: "states each other failure with its draws and its replay",
				body: func(c *prop.Case) {
					v := c.Draw(prop.Integer(0, 1000), drawn)
					if v%2 == 1 {
						fail(c, "odd")
					}
					if v >= 51 {
						fail(c, big)
					}
				},
				want: fmt.Sprintf(header, "counterexample", 1) +
					"\n  value: 1, 0 passes" +
					"\nfailure of odd: " +
					"\nreplay: prop.Replay(\"prop1:AAE\") or DOKIMI_ASSERT_PROP_REPLAY=prop1:AAE" +
					"\nother failure of big: " +
					"\n  value: 52" +
					"\nreplay: prop.Replay(\"prop1:ADQ\") or DOKIMI_ASSERT_PROP_REPLAY=prop1:ADQ",
			},
			{
				name: "states the requests of a body that diverges",
				body: diverges(prop.Integer(0, 9), prop.Boolean()),
				want: fmt.Sprintf(header, "flaky", 1) +
					"\ndivergence: the request at 0, recorded integer in [0, 9], replayed integer in [0, 1]",
			},
			{
				name: "states the end of a body that requests nothing where it requested before",
				body: diverges(prop.Integer(0, 9), prop.Just(0)),
				want: fmt.Sprintf(header, "flaky", 1) +
					"\ndivergence: the request at 0, recorded integer in [0, 9], replayed no request",
			},
			{
				name: "states the missing fingerprint of a replay",
				body: once(func(c *prop.Case) {
					c.Observe(7)
					fail(c, always)
				}),
				want: fmt.Sprintf(header, "flaky", 0) +
					"\nfailure of always: " +
					"\ndivergence: the fingerprint at 0, recorded 7, replayed no fingerprint",
			},
			{
				name: "states the failing case and the pass of a replay that passes",
				body: once(func(c *prop.Case) { fail(c, "once") }),
				want: fmt.Sprintf(header, "flaky", 0) +
					"\nfailure of once: " +
					"\ndivergence: the verdict at 0, recorded once, replayed a pass",
			},
			{
				name: "states a refuted coverage requirement",
				body: classifies(1000, even, isEven),
				opts: []prop.Option{prop.Require(even, 0.9)},
				want: fmt.Sprintf(header, "coverage-unmet", 100) +
					"\ncoverage: refuted, \"even\" counted 45 of 100 valid cases against a required share of 0.9",
			},
			{
				name: "states the counts of a run without a failing case",
				body: func(*prop.Case) {},
				want: fmt.Sprintf(header, "vacuous", 1),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				seat := &sentences{}
				prop.ForAll(seat, contract, tt.body, append(tt.opts, prop.Seed(7))...)
				assert.Equal(t, seat.all(), []string{tt.want}, "one sentence")
			})
		}

		t.Run("states a drawn value that contains itself with the cycle marked", func(t *testing.T) {
			t.Parallel()
			selfContaining := prop.Just(0).Map(func(int) map[string]any {
				m := map[string]any{}
				m["self"] = m
				return m
			})
			seat := &sentences{}
			prop.ForAll(seat, contract, func(c *prop.Case) {
				c.Draw(selfContaining, drawn)
				fail(c, always)
			}, prop.Seed(7))
			all := seat.all()
			assert.Length(t, all, 1, "one sentence")
			assert.Contains(t, all[0], "\n  value: map[self:<cycle>]", "the draw, with its cycle marked")
		})

		t.Run("states the assertion and the location of a failure with its sentence", func(t *testing.T) {
			t.Parallel()
			var at assert.Where
			body := func(c *prop.Case) {
				c.Draw(prop.Integer(0, 100), drawn)
				assert.True(c, false, "the flag is set"+here(&at))
			}
			seat := &sentences{}
			prop.ForAll(seat, contract, body, prop.Seed(7))
			want := fmt.Sprintf(header, "counterexample", 0) +
				"\n  value: 0, any value fails" +
				fmt.Sprintf("\nfailure of true at render_test.go:%d: the flag is set", at.Line) +
				replayZero
			assert.Equal(t, seat.all(), []string{want}, "one sentence")
		})
	})
}
