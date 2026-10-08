// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package conformance_test

import (
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"go.dokimi.dev/assert/conformance"
	"go.dokimi.dev/assert/internal/fault"
)

// TestBridge checks a bridge vector: the decoding of its generator from a
// fuzzer's bytes. Written with testing rather than with this library,
// because a verdict is not written with the subject.
func TestBridge(t *testing.T) {
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
				name: "returns nil for a vector that states the case of its bytes",
				give: bridged(digitGenerator, "07", `[7]`),
			},
			{
				name:       "returns a fault at the generator for a generator of an unknown id",
				give:       bridged(unknownGenerator, "07", `[7]`),
				wantPath:   inVector(fault.Field(generatorAt), fault.Field(genAt)),
				wantReason: noGenerator,
			},
			{
				name:       "returns a fault at the choices for choices other than the bytes decode to",
				give:       bridged(digitGenerator, "07", `[8]`),
				wantPath:   inVector(fault.Field(choicesAt)),
				wantReason: "the case records prop1:AAc, want [8]",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				expectFault(t, check(t, conformance.Bridge, tt.give), tt.wantPath, tt.wantReason)
			})
		}

		t.Run("returns a fault at the bytes for bytes that are no hexadecimal", func(t *testing.T) {
			t.Parallel()
			err := check(t, conformance.Bridge, bridged(digitGenerator, "0g", `[7]`))
			expectFault(t, err, inVector(fault.Field("bytes")), "the bytes are no hexadecimal")
			if _, ok := errors.AsType[hex.InvalidByteError](err); !ok {
				t.Fatalf("Check returns %v, want one caused by the invalid byte", err)
			}
		})
	})
}

// bridged returns a bridge vector of generator that decodes the hexadecimal
// bytes to choices and the integer 7.
func bridged(generator, bytes, choices string) string {
	return fmt.Sprintf(`{"generator":%s,"bytes":%q,"choices":%s,"value":{"type":"int","value":7},"rejected":false}`,
		generator, bytes, choices)
}
