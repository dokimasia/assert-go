// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !windows

package golden

// retried calls f once and returns its result. f reads a golden file or
// renames a file over one. No other handle on a file refuses its read or a
// rename over it on this platform.
func retried[T any](f func() (T, error)) (T, error) {
	return f()
}
