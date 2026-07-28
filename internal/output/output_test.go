package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ilyas-bkgo/devcheck/internal/checker"
)

func TestPrintJUnit(t *testing.T) {
	report := checker.NewReport([]checker.Result{
		{Name: "Go", Category: "tool", Passed: true},
		{Name: "PostgreSQL", Category: "service", Detail: "Port unreachable"},
	})
	var buffer bytes.Buffer
	if err := PrintJUnit(&buffer, report); err != nil {
		t.Fatal(err)
	}
	text := buffer.String()
	if !strings.Contains(text, `tests="2"`) || !strings.Contains(text, `failures="1"`) || !strings.Contains(text, "PostgreSQL") {
		t.Fatalf("unexpected JUnit output: %s", text)
	}
}

func TestPrintGitHubActions(t *testing.T) {
	var buffer bytes.Buffer
	PrintGitHubActions(&buffer, checker.NewReport([]checker.Result{{Name: "API", Category: "service", Detail: "HTTP 500"}}))
	if got := buffer.String(); !strings.Contains(got, "::error title=API::HTTP 500") {
		t.Fatalf("unexpected annotation: %s", got)
	}
}
