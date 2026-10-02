// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
)

// TestBridge checks a bridge vector: the decoding of its generator from a
// fuzzer's bytes.
func TestBridge(t *testing.T) {
	t.Parallel()

	t.Run("Check", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			give string
			want string
		}{
			{
				name: "returns nil for a vector that states the case of its bytes",
				give: bridged(digitGenerator, "07", `[7]`, `{"type":"int","value":7}`),
			},
			{
				name: "returns an error for a vector that is no JSON object",
				give: `[]`,
				want: "cannot unmarshal array",
			},
			{
				name: "returns an error for a generator of an unknown id",
				give: bridged(unknownGenerator, "07", `[7]`, `{"type":"int","value":7}`),
				want: `"widget" names no generator`,
			},
			{
				name: "returns an error for bytes that are no hexadecimal",
				give: bridged(digitGenerator, "0g", `[7]`, `{"type":"int","value":7}`),
				want: "invalid byte",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectCheck(t, check(t, conformance.Bridge, tt.give), tt.want)
			})
		}
	})
}

// bridged returns a bridge vector of generator that decodes the hexadecimal
// bytes to choices and the typed literal value.
func bridged(generator, bytes, choices, value string) string {
	return fmt.Sprintf(`{"generator":%s,"bytes":%q,"choices":%s,"value":%s,"rejected":false}`,
		generator, bytes, choices, value)
}
