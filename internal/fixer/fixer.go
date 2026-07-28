// Package fixer selects and executes configured repair commands.
package fixer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/ilyas-bkgo/devcheck/internal/checker"
)

// Options controls which fixes are offered and whether execution is disabled.
type Options struct {
	DryRun bool
	Only   map[string]bool
}

func Execute(command string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func ParseSelections(input string, maxLen int) []int {
	selected := make([]int, 0)
	for _, part := range strings.Split(input, ",") {
		var index int
		if _, err := fmt.Sscanf(strings.TrimSpace(part), "%d", &index); err == nil && index >= 1 && index <= maxLen {
			selected = append(selected, index-1)
		}
	}
	return selected
}

// Apply interactively offers and runs fixes for failed checks.
func Apply(failed []checker.Result, input io.Reader, output io.Writer) {
	ApplyWithOptions(failed, input, output, Options{})
}

// ApplyWithOptions interactively offers selected fixes, optionally as a dry run.
func ApplyWithOptions(failed []checker.Result, input io.Reader, output io.Writer, options Options) {
	fixable := make([]checker.Result, 0)
	for _, result := range failed {
		if strings.TrimSpace(result.Fix) != "" && (len(options.Only) == 0 || options.Only[result.Name]) {
			fixable = append(fixable, result)
		}
	}
	if len(fixable) == 0 {
		fmt.Fprintln(output, "\n🔧 Auto-Fix Mode: No executable fix commands available for failed checks.")
		return
	}

	if options.DryRun {
		fmt.Fprintln(output, "\n🔧 Running Auto-Fix Dry-Run Mode...")
	} else {
		fmt.Fprintln(output, "\n🔧 Running Auto-Fix Mode...")
	}
	fmt.Fprintln(output, "Available fixes for failed checks:")
	for i, result := range fixable {
		fmt.Fprintf(output, "  [%d] %-10s %-18s -> %s\n", i+1, "["+result.Category+"]", result.Name, result.Fix)
	}
	fmt.Fprintln(output, "\nSelect options:")
	fmt.Fprintln(output, "  a) Fix ALL")
	fmt.Fprintln(output, "  s) Select specific items")
	fmt.Fprintln(output, "  q) Quit / Skip")
	fmt.Fprint(output, "\nChoice [a/s/q]: ")

	reader := bufio.NewReader(input)
	choice, _ := reader.ReadString('\n')
	choice = strings.ToLower(strings.TrimSpace(choice))
	var selected []checker.Result
	switch choice {
	case "a", "all":
		selected = fixable
	case "s", "select":
		fmt.Fprint(output, "Enter item numbers to fix (comma-separated, e.g., 1, 2): ")
		values, _ := reader.ReadString('\n')
		for _, index := range ParseSelections(values, len(fixable)) {
			selected = append(selected, fixable[index])
		}
	case "q", "quit", "":
		fmt.Fprintln(output, "Auto-fix skipped.")
		return
	default:
		fmt.Fprintln(output, "Invalid choice. Skipping auto-fix.")
		return
	}
	if len(selected) == 0 {
		fmt.Fprintln(output, "No valid items selected. Skipping auto-fix.")
		return
	}

	fmt.Fprintln(output)
	for _, result := range selected {
		if options.DryRun {
			fmt.Fprintf(output, "Would execute fix for [%s] %s: %s\n", result.Category, result.Name, result.Fix)
			continue
		}
		fmt.Fprintf(output, "Executing fix for [%s] %s: %s\n", result.Category, result.Name, result.Fix)
		if err := Execute(result.Fix); err != nil {
			fmt.Fprintf(output, "❌ Fix failed: %v\n", err)
		} else {
			fmt.Fprintln(output, "✔ Fix executed successfully.")
		}
	}
}
