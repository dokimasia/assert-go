// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package nilcontextsafe

import (
	"context"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"
)

func TestFetch(t *testing.T) {
	assert.NilContextSafe(t, func(ctx context.Context) error { return fetch(ctx, "key") }, "Fetch accepts a nil context")
	assert.NotPanics(t, func() { _ = fetch(nil, "key") }, "Fetch accepts a nil context") // want `nil-context-safe: state the check with NilContextSafe`
	get := fetch
	expect.NotPanics(t, func() { _ = get(nil, string("key")) }, "Fetch accepts a nil context") // want `nil-context-safe: state the check with NilContextSafe`
	assert.NotPanics(t, func() { _ = count((*int)(nil)) }, "Count accepts a nil pointer")
	assert.NotPanics(t, func() { _ = same[context.Context](nil, t.Context()) }, "Same accepts a nil value")
	assert.NotPanics(t, program, "the program runs")
	_ = fetch(nil, "key")
	_ = fetch(t.Context(), "key")
}
