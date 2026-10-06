// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package expect

import "go.dokimi.dev/assert"

// Failure is what a failing assertion reports, the record of
// [go.dokimi.dev/assert]. Both surfaces report the same record. Want, Got
// and CaseFailure read its fields, as that package's Failure states.
type Failure = assert.Failure

// Where is the call site a failure came from.
type Where = assert.Where

// Reporter is a [TB] that takes the record rather than the sentence. aborting
// is false for every failure of this package.
type Reporter = assert.Reporter
