// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

//go:build !windows

package golden

// retried calls rename once and returns its error. rename renames a file
// over a golden file. No other handle on either file refuses the rename on
// this platform.
func retried(rename func() error) error {
	return rename()
}
