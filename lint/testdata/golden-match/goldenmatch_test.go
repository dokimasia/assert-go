// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: MIT

package goldenmatch

import (
	"bytes"
	"flag"
	"os"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
)

var rewrite = flag.Bool("update", false, "rewrite the golden files") // want `update-flag: state the check with golden.MatchAt and golden.ShouldUpdate`

var refresh bool

func init() {
	flag.BoolVar(&refresh, "update", false, "rewrite the golden files") // want `update-flag: state the check with golden.MatchAt and golden.ShouldUpdate`
}

var (
	verbose = flag.Bool("verbose", false, "log every case")
	label   = flag.String("update", "", "the label of the update")
	set     = flag.NewFlagSet("golden", flag.ContinueOnError)
	ofSet   = set.Bool("update", false, "rewrite the golden files")
	named   = flag.Bool(name(), false, "rewrite the golden files")
)

func name() string { return "update" }

func TestRender(t *testing.T) {
	flag.Parse()
	t.Log(*rewrite, refresh, *verbose, *label, *ofSet, *named, *update)
	golden.Match(t, "render.txt", render(), golden.ShouldUpdate())
	got := render()
	want, err := os.ReadFile("testdata/golden/render.txt")
	assert.NoError(t, err, "the golden file reads")
	assert.Equal(t, got, want, "Render writes the golden output") // want `golden-match: state the check with golden\.Match of os\.ReadFile\("testdata/golden/render\.txt"\)`
	if !bytes.Equal(render(), want) {                             // want `golden-match: state the check with golden.Match`
		t.Fatal("Render differs from the golden output")
	}
}
