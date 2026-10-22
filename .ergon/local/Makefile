# Copyright Dokimasia B.V. 2026
# SPDX-License-Identifier: MIT

# The targets of assert-go, which ergon init appends to the Makefile. spec vendors the definition of
# assert-spec into conformance, and the gate checks that the copy is intact.

##@ Repository

# The command that runs spec.
SPEC ?= spec

.PHONY: spec-sync spec-check

spec-sync: ## Vendor the definition of assert-spec at the head of its branch main, and the zone table that prop embeds
	$(SPEC) sync

spec-check: ## Check that the vendored definition is intact and comes from a commit of assert-spec
	$(SPEC) check

check: spec-check
