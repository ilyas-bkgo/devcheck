// Package output renders check reports for terminal and machine consumers.
package output

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ilyas-bkgo/devcheck/internal/checker"
	"golang.org/x/term"
)

func PrintJSON(writer io.Writer, report checker.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer, string(data))
	return err
}

type junitSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
}

// PrintJUnit renders a report in JUnit XML for CI test-report consumers.
func PrintJUnit(writer io.Writer, report checker.Report) error {
	suite := junitSuite{Name: "devcheck", Tests: len(report.Results)}
	for _, result := range report.Results {
		testCase := junitTestCase{Name: result.Name, ClassName: "devcheck." + result.Category}
		if !result.Passed {
			suite.Failures++
			testCase.Failure = &junitFailure{Message: result.Detail}
		}
		suite.Cases = append(suite.Cases, testCase)
	}
	data, err := xml.MarshalIndent(suite, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "%s\n", xml.Header+string(data))
	return err
}

// PrintGitHubActions emits workflow command annotations for failed checks.
func PrintGitHubActions(writer io.Writer, report checker.Report) {
	for _, result := range checker.Failed(report.Results) {
		fmt.Fprintf(writer, "::error title=%s::%s\n", githubEscape(result.Name), githubEscape(result.Detail))
	}
}

func githubEscape(value string) string {
	value = strings.ReplaceAll(value, "%", "%25")
	value = strings.ReplaceAll(value, "\r", "%0D")
	return strings.ReplaceAll(value, "\n", "%0A")
}

func PrintTerminal(writer io.Writer, report checker.Report, configPath string, quiet bool) {
	if quiet {
		if !report.Success {
			printSections(writer, checker.Failed(report.Results), false)
			fmt.Fprintln(writer, "\n❌ System health check failed.")
		}
		return
	}
	fmt.Fprintf(writer, "🔍 Running devcheck (%s)...\n\n", configPath)
	printSections(writer, report.Results, colorEnabled())
	if report.Success {
		fmt.Fprintln(writer, "\n✨ All checks passed!")
	} else {
		fmt.Fprintln(writer, "\n❌ System health check failed.")
	}
}

func colorEnabled() bool {
	return os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
}

func printSections(writer io.Writer, results []checker.Result, color bool) {
	sections := []struct{ category, title string }{
		{"tool", "🛠️  CLI Tools:"},
		{"path", "📁 Paths & Configurations:"},
		{"env", "🌐 Environment Variables:"},
		{"service", "🌐 Network Services:"},
	}
	printed := false
	for _, section := range sections {
		items := make([]checker.Result, 0)
		for _, result := range results {
			if result.Category == section.category {
				items = append(items, result)
			}
		}
		if len(items) == 0 {
			continue
		}
		if printed {
			fmt.Fprintln(writer)
		}
		printed = true
		fmt.Fprintln(writer, section.title)
		for _, result := range items {
			symbol := "✔"
			if !result.Passed {
				symbol = "✘"
			}
			if color {
				if result.Passed {
					symbol = "\033[32m✔\033[0m"
				} else {
					symbol = "\033[31m✘\033[0m"
				}
			}
			fmt.Fprintf(writer, " %s %-16s %s\n", symbol, result.Name, result.Detail)
		}
	}
}
