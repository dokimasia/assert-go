// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package history

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/literal"
)

// literals returns the typed literal of each value, as the detail of a call
// record states it, and an empty list for no value.
func literals[V any](values []V) []json.RawMessage {
	out := make([]json.RawMessage, len(values))
	for i, v := range values {
		out[i] = literal.Detail(v)
	}
	return out
}
