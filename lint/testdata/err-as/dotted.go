// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package erras

import (
	"errors"
	. "io/fs"
	"testing"

	"go.dokimi.dev/assert"
)

func dotted(t *testing.T, err error) {
	var pathErr *PathError
	assert.True(t, errors.As(err, &pathErr), "the error is a path error") // want `errors-as: state the check with ErrorAs`
	t.Log(pathErr.Path)
}
