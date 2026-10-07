// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package kind

import "example.test/lint/err-as/other"

type hidden struct{}

func (*hidden) Error() string { return "hidden" }

func Lookup() *other.NotFound { return nil }

func Hidden() *hidden { return nil }
