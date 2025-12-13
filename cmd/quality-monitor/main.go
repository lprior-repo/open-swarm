// Copyright (c) 2025 Open Swarm Contributors
//
// This software is released under the MIT License.
// See LICENSE file in the repository for details.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	minCoverageThreshold = 80.0
)

// QualityReport represents a comprehensive quality check report
type QualityReport struct {
	Timestamp       time.Time       `json:"timestamp"`
	TestResults     TestResults     `json:"test_results"`
	CoverageResults CoverageResults `json:"coverage_results"`
	LintResults     LintResults     `json:"lint_results"`
	FormatResults   FormatResults   `json:"format_results"`
	BeadsStatus     BeadsStatus     `json:"beads_status"`
	TODOCount       int             `json:"todo_count"`
	GoDocCoverage   map[string]bool `json:"godoc_coverage"`
	OverallStatus   string          `json:"overall_status"`
	Recommendations []string        `json:"recommendations"`
}

// TestResults holds test execution data
type TestResults struct {
	Passing        int      `json:"passing"`
	Failing        int      `json:"failing"`
	BuildErrors    int      `json:"build_errors"`
	TotalPackages  int      `json:"total_packages"`
	FailedPackages []string `json:"failed_packages"`
	Output         string   `json:"output"`
}

// CoverageResults holds coverage statistics
type CoverageResults struct {
	OverallCoverage float64            `json:"overall_coverage"`
	PackageCoverage map[string]float64 `json:"package_coverage"`
	Below80Percent  []string           `json:"below_80_percent"`
	ZeroCoverage    []string           `json:"zero_coverage"`
}

// LintResults holds linting information
type LintResults struct {
	Passing        bool     `json:"passing"`
	IssueCount     int      `json:"issue_count"`
	CriticalIssues []string `json:"critical_issues"`
	Output         string   `json:"output"`
}

// FormatResults holds formatting check results
type FormatResults struct {
	Passing          bool     `json:"passing"`
	UnformattedFiles []string `json:"unformatted_files"`
}

// BeadsStatus holds Beads task tracking data
type BeadsStatus struct {
	InProgress int      `json:"in_progress"`
	Open       int      `json:"open"`
	TaskIDs    []string `json:"task_ids"`
}

func main() {
	workDir := "/home/lewis/src/open-swarm"

	if err := os.Chdir(workDir); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to change directory: %v\n", err)
		os.Exit(1)
	}

	report := &QualityReport{
		Timestamp: time.Now(),
		CoverageResults: CoverageResults{
			PackageCoverage: make(map[string]float64),
		},
		GoDocCoverage: make(map[string]bool),
	}

	fmt.Println("========================================")
	fmt.Println("Open Swarm Quality Monitor")
	fmt.Printf("Time: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println("========================================")
	fmt.Println()

	// Run all checks
	runTestSuite(report)
	runCoverageCheck(report)
	runFormatCheck(report)
	runLintCheck(report)
	checkBeadsStatus(report)
	checkTODOs(report)
	checkGoDocCoverage(report)

	// Generate recommendations
	generateRecommendations(report)

	// Determine overall status
	determineOverallStatus(report)

	// Print summary
	printSummary(report)

	// Save JSON report
	saveReport(report)

	// Exit with appropriate code
	if report.OverallStatus != "PASSING" {
		os.Exit(1)
	}
}

func runTestSuite(report *QualityReport) {
	fmt.Println("--- Running Test Suite ---")

	cmd := exec.Command("go", "test", "./...")
	output, err := cmd.CombinedOutput()

	report.TestResults.Output = string(output)

	// Parse output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "ok"):
			report.TestResults.Passing++
		case strings.HasPrefix(line, "FAIL"):
			report.TestResults.Failing++
			// Extract package name
			parts := strings.Fields(line)
			if len(parts) > 1 {
				report.TestResults.FailedPackages = append(report.TestResults.FailedPackages, parts[1])
			}
		case strings.Contains(line, "[build failed]"):
			report.TestResults.BuildErrors++
		}
	}

	report.TestResults.TotalPackages = report.TestResults.Passing + report.TestResults.Failing

	if err != nil {
		fmt.Printf("❌ Tests FAILING (%d passing, %d failing, %d build errors)\n",
			report.TestResults.Passing, report.TestResults.Failing, report.TestResults.BuildErrors)
	} else {
		fmt.Printf("✅ Tests PASSING (%d packages)\n", report.TestResults.Passing)
	}
	fmt.Println()
}

func runCoverageCheck(report *QualityReport) {
	fmt.Println("--- Checking Test Coverage ---")

	cmd := exec.Command("go", "test", "-cover", "./...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("⚠️  Some packages failed coverage check")
	}

	// Parse coverage output
	coverageRegex := regexp.MustCompile(`coverage:\s+([\d.]+)%\s+of\s+statements`)
	packageRegex := regexp.MustCompile(`^(ok|FAIL)\s+([^\s]+)`)

	lines := strings.Split(string(output), "\n")
	var totalCoverage float64
	var packageCount int

	for i, line := range lines {
		if matches := packageRegex.FindStringSubmatch(line); matches != nil {
			pkg := matches[2]

			// Look for coverage on this or next line
			checkLine := line
			if i+1 < len(lines) {
				checkLine = line + " " + lines[i+1]
			}

			if covMatches := coverageRegex.FindStringSubmatch(checkLine); covMatches != nil {
				coverage := 0.0
				if _, err := fmt.Sscanf(covMatches[1], "%f", &coverage); err != nil {
					log.Printf("Warning: failed to parse coverage: %v", err)
				}

				report.CoverageResults.PackageCoverage[pkg] = coverage
				totalCoverage += coverage
				packageCount++

				if coverage == 0 {
					report.CoverageResults.ZeroCoverage = append(report.CoverageResults.ZeroCoverage, pkg)
				} else if coverage < minCoverageThreshold {
					report.CoverageResults.Below80Percent = append(report.CoverageResults.Below80Percent, pkg)
				}
			}
		}
	}

	if packageCount > 0 {
		report.CoverageResults.OverallCoverage = totalCoverage / float64(packageCount)
	}

	fmt.Printf("Overall Coverage: %.1f%%\n", report.CoverageResults.OverallCoverage)
	fmt.Printf("Packages with 0%% coverage: %d\n", len(report.CoverageResults.ZeroCoverage))
	fmt.Printf("Packages below 80%% coverage: %d\n", len(report.CoverageResults.Below80Percent))
	fmt.Println()
}

func runFormatCheck(report *QualityReport) {
	fmt.Println("--- Checking Formatting ---")

	cmd := exec.Command("gofmt", "-l", ".")
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("⚠️  gofmt command failed: %v\n", err)
		return
	}

	unformatted := strings.TrimSpace(string(output))
	if unformatted == "" {
		report.FormatResults.Passing = true
		fmt.Println("✅ All files properly formatted")
	} else {
		report.FormatResults.Passing = false
		report.FormatResults.UnformattedFiles = strings.Split(unformatted, "\n")
		fmt.Printf("❌ %d files need formatting\n", len(report.FormatResults.UnformattedFiles))
	}
	fmt.Println()
}

func runLintCheck(report *QualityReport) {
	fmt.Println("--- Running Linter ---")

	lintCmd := filepath.Join(os.Getenv("HOME"), "go", "bin", "golangci-lint")
	// #nosec G204 - lintCmd path is constructed from hardcoded string and trusted HOME environment
	cmd := exec.Command(lintCmd, "run", "--timeout=5m")
	output, err := cmd.CombinedOutput()

	report.LintResults.Output = string(output)

	// Count issues
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ".go:") && !strings.HasPrefix(line, "level=") {
			report.LintResults.IssueCount++

			// Check for critical issues
			if strings.Contains(line, "cannot use") ||
				strings.Contains(line, "undefined:") ||
				strings.Contains(line, "type error") {
				report.LintResults.CriticalIssues = append(report.LintResults.CriticalIssues, line)
			}
		}
	}

	if err == nil && report.LintResults.IssueCount == 0 {
		report.LintResults.Passing = true
		fmt.Println("✅ Linting passed with no issues")
	} else {
		report.LintResults.Passing = false
		fmt.Printf("❌ Linting found %d issues (%d critical)\n",
			report.LintResults.IssueCount, len(report.LintResults.CriticalIssues))
	}
	fmt.Println()
}

func checkBeadsStatus(report *QualityReport) {
	fmt.Println("--- Checking Beads Tasks ---")

	cmd := exec.Command("bd", "list", "--status", "in_progress", "--json")
	output, err := cmd.CombinedOutput()

	if err == nil && len(output) > 0 {
		var tasks []map[string]interface{}
		if err := json.Unmarshal(output, &tasks); err == nil {
			report.BeadsStatus.InProgress = len(tasks)
			for _, task := range tasks {
				if id, ok := task["id"].(string); ok {
					report.BeadsStatus.TaskIDs = append(report.BeadsStatus.TaskIDs, id)
				}
			}
		}
	}

	cmd = exec.Command("bd", "list", "--status", "open", "--json")
	output, err = cmd.CombinedOutput()

	if err == nil && len(output) > 0 {
		var tasks []map[string]interface{}
		if err := json.Unmarshal(output, &tasks); err == nil {
			report.BeadsStatus.Open = len(tasks)
		}
	}

	fmt.Printf("In Progress: %d tasks\n", report.BeadsStatus.InProgress)
	fmt.Printf("Open: %d tasks\n", report.BeadsStatus.Open)
	fmt.Println()
}

func checkTODOs(report *QualityReport) {
	fmt.Println("--- Checking for TODOs/FIXMEs ---")

	cmd := exec.Command("grep", "-r", "TODO\\|FIXME",
		"--include=*.go", "internal/", "pkg/", "cmd/")
	output, _ := cmd.CombinedOutput()

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" && !strings.Contains(line, "_test.go") {
			report.TODOCount++
		}
	}

	if report.TODOCount == 0 {
		fmt.Println("✅ No TODO/FIXME comments in critical code")
	} else {
		fmt.Printf("⚠️  Found %d TODO/FIXME comments\n", report.TODOCount)
	}
	fmt.Println()
}

func checkGoDocCoverage(report *QualityReport) {
	fmt.Println("--- Checking GoDoc Coverage ---")

	// Check key packages
	packages := []string{"pkg/agent", "pkg/coordinator", "internal/config"}

	for _, pkg := range packages {
		cmd := exec.Command("go", "doc", "-all", "./"+pkg)
		output, err := cmd.CombinedOutput()

		// Simple check: if go doc works and has content, consider it documented
		report.GoDocCoverage[pkg] = (err == nil && len(output) > 100)
	}

	documented := 0
	for _, hasDoc := range report.GoDocCoverage {
		if hasDoc {
			documented++
		}
	}

	fmt.Printf("Documented packages: %d/%d\n", documented, len(packages))
	fmt.Println()
}

func generateRecommendations(report *QualityReport) {
	if report.TestResults.BuildErrors > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("CRITICAL: Fix %d build errors blocking test execution", report.TestResults.BuildErrors))
	}

	if report.TestResults.Failing > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("HIGH: Fix %d failing test packages", report.TestResults.Failing))
	}

	if len(report.LintResults.CriticalIssues) > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("CRITICAL: Resolve %d critical linting issues", len(report.LintResults.CriticalIssues)))
	}

	if report.CoverageResults.OverallCoverage < minCoverageThreshold {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("MEDIUM: Increase test coverage from %.1f%% to 80%%", report.CoverageResults.OverallCoverage))
	}

	if len(report.CoverageResults.ZeroCoverage) > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("HIGH: Add tests for %d packages with 0%% coverage", len(report.CoverageResults.ZeroCoverage)))
	}

	if !report.FormatResults.Passing {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("LOW: Format %d files with gofmt", len(report.FormatResults.UnformattedFiles)))
	}

	if report.BeadsStatus.InProgress > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("INFO: Monitor %d in-progress tasks for completion", report.BeadsStatus.InProgress))
	}
}

func determineOverallStatus(report *QualityReport) {
	switch {
	case report.TestResults.BuildErrors > 0 ||
		report.TestResults.Failing > 0 ||
		len(report.LintResults.CriticalIssues) > 0:
		report.OverallStatus = "FAILING"
	case report.CoverageResults.OverallCoverage < minCoverageThreshold ||
		!report.FormatResults.Passing ||
		report.LintResults.IssueCount > 0:
		report.OverallStatus = "NEEDS_IMPROVEMENT"
	default:
		report.OverallStatus = "PASSING"
	}
}

func printSummary(report *QualityReport) {
	fmt.Println("========================================")
	fmt.Println("QUALITY REPORT SUMMARY")
	fmt.Println("========================================")
	fmt.Printf("Overall Status: %s\n\n", report.OverallStatus)

	fmt.Printf("Tests: %d passing, %d failing, %d build errors\n",
		report.TestResults.Passing, report.TestResults.Failing, report.TestResults.BuildErrors)
	fmt.Printf("Coverage: %.1f%% (Target: 80%%)\n", report.CoverageResults.OverallCoverage)
	fmt.Printf("Linting: %d issues (%d critical)\n",
		report.LintResults.IssueCount, len(report.LintResults.CriticalIssues))
	fmt.Printf("Formatting: %s\n", formatStatus(report.FormatResults.Passing))
	fmt.Printf("Beads: %d in-progress, %d open\n",
		report.BeadsStatus.InProgress, report.BeadsStatus.Open)

	if len(report.Recommendations) > 0 {
		fmt.Println("\nRecommendations:")
		for i, rec := range report.Recommendations {
			fmt.Printf("%d. %s\n", i+1, rec)
		}
	}

	fmt.Println("========================================")
}

func formatStatus(passing bool) string {
	if passing {
		return "PASS"
	}
	return "FAIL"
}

func saveReport(report *QualityReport) {
	reportDir := "/tmp/quality-reports"
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		fmt.Printf("Failed to create report directory: %v\n", err)
		return
	}

	filename := filepath.Join(reportDir,
		fmt.Sprintf("quality-report-%s.json", report.Timestamp.Format("2006-01-02-150405")))

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal report: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write report: %v\n", err)
		return
	}

	fmt.Printf("\nReport saved to: %s\n", filename)
}
