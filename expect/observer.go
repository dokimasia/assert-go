// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import "go.dokimi.dev/assert"

// Observer is a [TB] that observes each assertion call, the observer of
// [go.dokimi.dev/assert]. The calls of this surface reach it as the calls of
// the aborting surface do.
type Observer = assert.Observer

// Call is one assertion call that an [Observer] receives, the call of
// [go.dokimi.dev/assert].
type Call = assert.Call
