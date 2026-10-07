// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package nilcontextsafe

import "context"

func fetch(ctx context.Context, key string) error { return nil }

func count(values *int) int { return 0 }

func same[T comparable](a, b T) bool { return a == b }

func program() {
	_ = fetch(nil, "key")
}
