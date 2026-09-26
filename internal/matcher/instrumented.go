// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

//go:build race || msan || asan

package matcher

// instrumented reports whether the race detector, msan or asan
// instruments this build. It does.
const instrumented = true
