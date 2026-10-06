// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import "go.dokimi.dev/assert"

// TB is the seat that an assertion reports through, the seat of
// [go.dokimi.dev/assert]. [testing.T], [testing.B] and [Recorder] satisfy
// it.
type TB = assert.TB
