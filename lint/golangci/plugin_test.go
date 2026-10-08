// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golangci_test

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"

	"go.dokimi.dev/assert/lint"
	"go.dokimi.dev/assert/lint/golangci"
)

func TestPlugin(t *testing.T) {
	t.Parallel()

	t.Run("New", func(t *testing.T) {
		t.Parallel()

		t.Run("is the constructor that golangci-lint finds under assertlint", func(t *testing.T) {
			t.Parallel()
			newPlugin, err := register.GetPlugin("assertlint")
			if err != nil {
				t.Fatalf("register.GetPlugin(%q) returns %v, want a constructor", "assertlint", err)
			}
			p, err := newPlugin(nil)
			if err != nil {
				t.Fatalf("the constructor returns %v for no settings, want a plugin", err)
			}
			analyzed(t, p)
		})

		tests := []struct {
			name    string
			give    any
			wantErr bool
		}{
			{"returns a plugin for no settings", nil, false},
			{"returns a plugin for an empty map of settings", map[string]any{}, false},
			{"returns an error for a setting", map[string]any{"rules": []any{"compare"}}, true},
			{"returns an error for settings that are no map", "compare", true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				p, err := golangci.New(tt.give)
				if tt.wantErr {
					if err == nil || p != nil {
						t.Fatalf("New(%v) returns %v and %v, want no plugin and an error", tt.give, p, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("New(%v) returns %v, want a plugin", tt.give, err)
				}
				analyzed(t, p)
			})
		}
	})

	t.Run("BuildAnalyzers", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the analyzer of go.dokimi.dev/assert/lint alone", func(t *testing.T) {
			t.Parallel()
			p, err := golangci.New(nil)
			if err != nil {
				t.Fatalf("New(nil) returns %v, want a plugin", err)
			}
			analyzed(t, p)
		})
	})

	t.Run("GetLoadMode", func(t *testing.T) {
		t.Parallel()

		t.Run("asks for the types of each package", func(t *testing.T) {
			t.Parallel()
			p, err := golangci.New(nil)
			if err != nil {
				t.Fatalf("New(nil) returns %v, want a plugin", err)
			}
			if got := p.GetLoadMode(); got != register.LoadModeTypesInfo {
				t.Errorf("GetLoadMode() = %q, want %q", got, register.LoadModeTypesInfo)
			}
		})
	})
}

// analyzed fails the test unless p builds the analyzer of
// go.dokimi.dev/assert/lint and no other.
func analyzed(t *testing.T, p register.LinterPlugin) {
	t.Helper()
	analyzers, err := p.BuildAnalyzers()
	if err != nil || len(analyzers) != 1 || analyzers[0] != lint.Analyzer {
		t.Errorf("BuildAnalyzers() returns %v and %v, want lint.Analyzer alone", analyzers, err)
	}
}
