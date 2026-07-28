package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ilyas-bkgo/devcheck/internal/checker"
	"github.com/ilyas-bkgo/devcheck/internal/config"
	"github.com/ilyas-bkgo/devcheck/internal/doctor"
	"github.com/ilyas-bkgo/devcheck/internal/fixer"
	"github.com/ilyas-bkgo/devcheck/internal/output"
	"github.com/ilyas-bkgo/devcheck/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			os.Exit(runInit(os.Args[2:]))
		case "version", "--version":
			printVersion()
			return
		case "doctor":
			os.Exit(runDoctor(os.Args[2:]))
		case "validate":
			os.Exit(runValidate(os.Args[2:]))
		}
	}

	var configPath, profile, format, fixOnly string
	var jsonOutput, quietOutput, fixMode, fixDryRun bool
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	flag.BoolVar(&jsonOutput, "json", false, "Output results in JSON format")
	flag.BoolVar(&jsonOutput, "j", false, "Output results in JSON format (shorthand)")
	flag.BoolVar(&quietOutput, "quiet", false, "Only print failed checks")
	flag.BoolVar(&quietOutput, "q", false, "Only print failed checks (shorthand)")
	flag.BoolVar(&fixMode, "fix", false, "Prompt and execute defined fixes for failed checks")
	flag.BoolVar(&fixMode, "f", false, "Prompt and execute defined fixes for failed checks (shorthand)")
	flag.StringVar(&profile, "profile", "", "Configuration profile to add")
	flag.StringVar(&format, "format", "terminal", "Output format: terminal, json, junit, or github-actions")
	flag.BoolVar(&fixDryRun, "fix-dry-run", false, "Show selected fixes without executing them")
	flag.StringVar(&fixOnly, "fix-only", "", "Only offer fixes for named checks (comma-separated)")
	flag.Parse()
	if jsonOutput {
		format = "json"
	}
	if !validFormat(format) {
		fmt.Printf("❌ Unknown output format %q. Use terminal, json, junit, or github-actions.\n", format)
		os.Exit(2)
	}
	if fixDryRun || fixOnly != "" {
		fixMode = true
	}

	resolvedConfig := config.ResolvePath(configPath)
	cfg, err := config.LoadProfile(resolvedConfig, profile)
	if err != nil {
		if format == "json" {
			_ = output.PrintJSON(os.Stdout, checker.Report{Success: false})
		} else {
			fmt.Printf("❌ Could not load %s: %v\n", resolvedConfig, err)
		}
		os.Exit(2)
	}
	if err := config.Validate(cfg); err != nil {
		if format == "json" {
			_ = output.PrintJSON(os.Stdout, checker.Report{Success: false})
		} else {
			fmt.Printf("❌ Invalid configuration: %v\n", err)
		}
		os.Exit(2)
	}

	report := checker.NewReport(checker.Run(cfg))
	switch format {
	case "json":
		_ = output.PrintJSON(os.Stdout, report)
	case "junit":
		_ = output.PrintJUnit(os.Stdout, report)
	case "github-actions":
		output.PrintGitHubActions(os.Stdout, report)
	default:
		output.PrintTerminal(os.Stdout, report, resolvedConfig, quietOutput)
	}
	if fixMode && !report.Success {
		fixer.ApplyWithOptions(checker.Failed(report.Results), os.Stdin, os.Stdout, fixer.Options{DryRun: fixDryRun, Only: splitNames(fixOnly)})
	}
	if !report.Success {
		os.Exit(1)
	}
}

func validFormat(format string) bool {
	return format == "terminal" || format == "json" || format == "junit" || format == "github-actions"
}

func splitNames(value string) map[string]bool {
	if value == "" {
		return nil
	}
	names := make(map[string]bool)
	for _, name := range strings.Split(value, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names[name] = true
		}
	}
	return names
}

func runInit(args []string) int {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	template := flags.String("template", "default", "Starter template: default, node, python, or go")
	flags.StringVar(template, "t", "default", "Starter template shorthand")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := config.WriteStarterTemplate("devcheck.yaml", *template); err != nil {
		switch err {
		case config.ErrConfigExists:
			fmt.Println("devcheck.yaml already exists!")
		case config.ErrUnknownTemplate:
			fmt.Printf("Unknown template %q. Available templates: %s\n", *template, strings.Join(config.TemplateNames(), ", "))
		default:
			fmt.Printf("Failed to create devcheck.yaml: %v\n", err)
		}
		return 1
	}
	fmt.Printf("Created %s starter config at ./devcheck.yaml! Run 'devcheck' to test it.\n", *template)
	return 0
}

func runDoctor(args []string) int {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	var configPath, profile string
	flags.StringVar(&configPath, "config", "", "Path to config file")
	flags.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	flags.StringVar(&profile, "profile", "", "Configuration profile to validate")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if doctor.Run(os.Stdout, configPath, profile) {
		return 0
	}
	return 1
}

func runValidate(args []string) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(os.Stdout)
	var configPath, profile string
	flags.StringVar(&configPath, "config", "", "Path to config file")
	flags.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	flags.StringVar(&profile, "profile", "", "Configuration profile to validate")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	path := config.ResolvePath(configPath)
	cfg, err := config.LoadProfile(path, profile)
	if err == nil {
		err = config.Validate(cfg)
	}
	if err != nil {
		fmt.Printf("❌ Invalid configuration: %v\n", err)
		return 2
	}
	fmt.Printf("✔ Configuration is valid: %s\n", path)
	return 0
}

func printVersion() {
	info := version.Current()
	fmt.Printf("devcheck %s\ncommit: %s\nbuilt: %s\nexecutable: %s\nruntime: %s %s/%s\n", info.Version, info.Commit, info.Date, info.Executable, info.GoVersion, info.OS, info.Arch)
}
