package fixer

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/ilyas-bkgo/devcheck/internal/checker"
)

func TestParseSelections(t *testing.T) {
	if got, want := ParseSelections("0, 1, foo, 3, 4", 3), []int{0, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestApplyWithOptionsDryRun(t *testing.T) {
	var output bytes.Buffer
	ApplyWithOptions([]checker.Result{{Name: "Missing", Category: "tool", Fix: "echo fixed"}}, strings.NewReader("a\n"), &output, Options{DryRun: true})
	if !strings.Contains(output.String(), "Would execute fix") {
		t.Fatalf("unexpected output: %s", output.String())
	}
}
