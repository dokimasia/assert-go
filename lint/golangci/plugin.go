// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package golangci

import (
	"fmt"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"go.dokimi.dev/assert/lint"
)

// init registers New under the name assertlint, which golangci-lint looks
// up for the linter of that name in its configuration.
func init() {
	register.Plugin("assertlint", New)
}

// New returns the plugin, from the settings of the linter in the
// configuration of golangci-lint. The analyzer takes no settings, so New
// returns an error for any setting.
func New(settings any) (register.LinterPlugin, error) {
	if _, err := register.DecodeSettings[struct{}](settings); err != nil {
		return nil, fmt.Errorf("golangci: assertlint takes no settings: %w", err)
	}
	return plugin{}, nil
}

// plugin is the module plugin assertlint.
type plugin struct{}

// BuildAnalyzers returns the analyzer of go.dokimi.dev/assert/lint alone.
func (plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{lint.Analyzer}, nil
}

// GetLoadMode asks golangci-lint for the types of each package, which the
// rules read.
func (plugin) GetLoadMode() string { return register.LoadModeTypesInfo }
