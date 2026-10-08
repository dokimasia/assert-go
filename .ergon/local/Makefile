# Copyright Dokimasia B.V. 2026
# SPDX-License-Identifier: MIT

# The targets of assert-go, which ergon init appends to the Makefile. The gate also checks that the
# vendored definition of assert-spec is intact.

##@ Repository

.PHONY: spec-sync spec-check

spec-sync: ## Refresh the vendored definition from assert-spec, and the zone table that prop embeds
	@./tools/spec-sync.sh conformance/spec go
	@cp conformance/spec/zones.json internal/prop/zone/zones.json

spec-check: ## Check that the vendored definition is intact, and report when it is behind
	@./tools/spec-check.sh conformance/spec

check: spec-check
