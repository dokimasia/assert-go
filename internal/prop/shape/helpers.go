// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package shape

import (
	"encoding/json"

	"go.dokimi.dev/assert/internal/fault"
	"go.dokimi.dev/assert/internal/prop/engine"
)

// unreadable returns a fault of the kind ErrShape whose reason is format
// with args, as fmt.Sprintf formats them.
func unreadable(format string, args ...any) *fault.Error {
	return fault.Of(ErrShape, format, args...)
}

// uninvertible returns a fault of the kind engine.ErrCannotInvert whose
// reason is format with args, as fmt.Sprintf formats them.
func uninvertible(format string, args ...any) *fault.Error {
	return fault.Of(engine.ErrCannotInvert, format, args...)
}

// show returns the JSON text of v, for a reason that names it.
func show(v any) string {
	text, _ := json.Marshal(v)
	return string(text)
}
