// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package lint_test

import "testing"

func TestFiles(t *testing.T) {
	t.Parallel()
	t.Run("Analyzer", func(t *testing.T) {
		t.Parallel()
		tests := []fixture{
			{name: "reports a check that a path does not exist as files.Absent", give: "./path-absent"},
			{name: "reports IsDir of a file's information as files.IsDir", give: "./is-dir"},
			{name: "reports IsRegular of a file's mode as files.IsFile", give: "./is-file"},
			{name: "reports an equality of a file's permission bits as files.HasMode", give: "./has-mode"},
			{name: "reports an equality of a link's target as files.LinksTo", give: "./links-to"},
		}
		analyzeEach(t, tests)
	})
}
