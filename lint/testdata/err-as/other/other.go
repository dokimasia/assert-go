// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package other

type NotFound struct{}

func (*NotFound) Error() string { return "not found" }
