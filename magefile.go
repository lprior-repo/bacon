//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

// Build compiles all Lambda functions with optimized parallelism and caching
func Build() error {
	mg.Deps(Clean, ModTidy) // Ensure dependencies are ready

	lambdaFunctions, err := findLambdaFunctions()
	if err != nil {
		return fmt.Errorf("failed to find Lambda functions: %w", err)
	}

	if len(lambdaFunctions) == 0 {
		fmt.Println("No Lambda functions found to build")
		return nil
	}

	fmt.Printf("Building %d Lambda functions with optimized parallelism...\n", len(lambdaFunctions))

	// Optimize concurrency based on system resources and I/O constraints
	maxConcurrency := min(runtime.NumCPU()*2, len(lambdaFunctions)) // Allow more concurrency for I/O bound operations
	
	// Skip prewarming to avoid directory race conditions
	// Dependencies will be downloaded during build if needed

	// Build with work-stealing pool for better load balancing
	return buildWithWorkStealing(lambdaFunctions, maxConcurrency)
}

// buildWithWorkStealing implements a work-stealing pool for optimal resource utilization
func buildWithWorkStealing(functions []LambdaFunction, maxWorkers int) error {
	workQueue := make(chan LambdaFunction, len(functions))
	results := make(chan error, len(functions))
	
	// Start workers
	for i := 0; i < maxWorkers; i++ {
		go buildWorker(i, workQueue, results)
	}
	
	// Queue all work
	for _, fn := range functions {
		workQueue <- fn
	}
	close(workQueue)
	
	// Collect results
	var errors []error
	for i := 0; i < len(functions); i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		fmt.Printf("❌ %d builds failed:\n", len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("build failed with %d errors", len(errors))
	}
	
	fmt.Printf("🎉 Successfully built all %d Lambda functions\n", len(functions))
	return nil
}

// buildWorker processes Lambda builds from the work queue
func buildWorker(id int, workQueue <-chan LambdaFunction, results chan<- error) {
	for fn := range workQueue {
		start := time.Now()
		err := buildLambdaFunction(fn)
		duration := time.Since(start)
		
		if err != nil {
			fmt.Printf("❌ Worker %d: Failed to build %s (%v)\n", id, fn.Name, duration)
			results <- fmt.Errorf("failed to build %s: %w", fn.Name, err)
		} else {
			fmt.Printf("✅ Worker %d: Built %s (%v)\n", id, fn.Name, duration)
			results <- nil
		}
	}
}

// prewarmBuildCache downloads all dependencies to warm the build cache
func prewarmBuildCache() error {
	fmt.Println("🔥 Prewarming build cache...")
	
	modules, err := findGoModules()
	if err != nil {
		return err
	}
	
	// Download dependencies in parallel
	var wg sync.WaitGroup
	maxConcurrency := runtime.NumCPU()
	semaphore := make(chan struct{}, maxConcurrency)
	
	for _, module := range modules {
		wg.Add(1)
		go func(modPath string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			dir := filepath.Dir(modPath)
			oldDir, _ := os.Getwd()
			os.Chdir(dir)
			defer os.Chdir(oldDir)
			
			// Download and verify dependencies
			sh.Run("go", "mod", "download", "-x") // -x for verbose output
		}(module)
	}
	
	wg.Wait()
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Clean removes all build artifacts
func Clean() error {
	fmt.Println("🧹 Cleaning build artifacts...")
	
	// Remove Go build cache
	if err := sh.Run("go", "clean", "-cache"); err != nil {
		return fmt.Errorf("failed to clean Go cache: %w", err)
	}

	// Remove any existing binaries in Lambda directories
	lambdaFunctions, err := findLambdaFunctions()
	if err != nil {
		return fmt.Errorf("failed to find Lambda functions: %w", err)
	}

	for _, function := range lambdaFunctions {
		binaryPath := filepath.Join(function.Path, "main")
		if _, err := os.Stat(binaryPath); err == nil {
			if err := os.Remove(binaryPath); err != nil {
				fmt.Printf("Warning: failed to remove %s: %v\n", binaryPath, err)
			}
		}
	}

	fmt.Println("✅ Clean completed")
	return nil
}

// Test runs unit tests for all packages in parallel with coverage
func Test() error {
	fmt.Println("🧪 Running unit tests in parallel...")
	
	// Run tests with parallel execution, race detection, and coverage
	env := map[string]string{
		"CGO_ENABLED": "1", // Required for race detector
	}
	
	args := []string{
		"test",
		"-race",
		"-parallel", fmt.Sprintf("%d", runtime.NumCPU()),
		"-coverprofile=coverage.out",
		"-covermode=atomic",
		"-v",
		"./...",
	}
	
	return sh.RunWith(env, "go", args...)
}

// TestUnit runs unit tests only (excludes integration tests)
func TestUnit() error {
	fmt.Println("🧪 Running unit tests only...")
	
	env := map[string]string{
		"CGO_ENABLED": "1",
	}
	
	args := []string{
		"test",
		"-race",
		"-parallel", fmt.Sprintf("%d", runtime.NumCPU()),
		"-short", // Skip integration tests
		"-coverprofile=coverage.out",
		"-covermode=atomic",
		"-v",
		"./...",
	}
	
	return sh.RunWith(env, "go", args...)
}

// TestContract runs contract tests against Terraform configurations
func TestContract() error {
	fmt.Println("🔗 Running contract tests...")
	
	env := map[string]string{
		"CGO_ENABLED": "1",
	}
	
	args := []string{
		"test",
		"-tags", "contract",
		"-parallel", fmt.Sprintf("%d", runtime.NumCPU()),
		"-v",
		"./tests/contract/...",
	}
	
	return sh.RunWith(env, "go", args...)
}

// TestE2E runs end-to-end tests against ephemeral infrastructure
func TestE2E() error {
	fmt.Println("🌐 Running end-to-end tests...")
	
	env := map[string]string{
		"CGO_ENABLED": "1",
	}
	
	args := []string{
		"test",
		"-tags", "e2e",
		"-timeout", "30m", // E2E tests can take longer
		"-v",
		"./tests/e2e/...",
	}
	
	return sh.RunWith(env, "go", args...)
}

// TestMutation runs mutation testing on all Go modules with 95% threshold
func TestMutation() error {
	fmt.Println("🧬 Running mutation testing with 95% threshold...")
	
	// Ensure go-mutesting is installed
	if err := ensureMutationTester(); err != nil {
		return err
	}
	
	// Create output directory
	outputDir := "mutation-results"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Find all Go modules with tests
	modules, err := findModulesWithTests()
	if err != nil {
		return fmt.Errorf("failed to find modules with tests: %w", err)
	}
	
	if len(modules) == 0 {
		fmt.Println("⚠️  No modules with tests found for mutation testing")
		return nil
	}
	
	fmt.Printf("🔬 Found %d modules with tests to mutate\n", len(modules))
	
	// Run mutation testing in parallel
	return runMutationTests(modules, outputDir)
}

// TestMutationModule runs mutation testing on a specific module
func TestMutationModule(module string) error {
	fmt.Printf("🧬 Running mutation testing on module: %s\n", module)
	
	if err := ensureMutationTester(); err != nil {
		return err
	}
	
	// Check if module exists and has tests
	if !hasTests(module) {
		return fmt.Errorf("module %s does not exist or has no tests", module)
	}
	
	outputDir := filepath.Join("mutation-results", filepath.Base(module))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	return runSingleModuleMutation(module, outputDir)
}

// Lint runs linters on all Go code
func Lint() error {
	fmt.Println("🔍 Running linters...")
	
	// First try the GOPATH/bin location
	golangciLint := filepath.Join(os.Getenv("GOPATH"), "bin", "golangci-lint")
	
	// Check if golangci-lint is available in GOPATH/bin
	if err := sh.Run(golangciLint, "--version"); err != nil {
		fmt.Printf("golangci-lint not found in %s, trying system PATH...\n", golangciLint)
		
		// Try system PATH
		if err := sh.Run("golangci-lint", "--version"); err != nil {
			fmt.Println("golangci-lint not found anywhere, installing latest version...")
			if err := sh.Run("go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"); err != nil {
				return fmt.Errorf("failed to install golangci-lint: %w", err)
			}
			// Use GOPATH version after install
			return sh.Run(golangciLint, "run", "./...")
		}
		// Use system PATH version
		return sh.Run("golangci-lint", "run", "./...")
	}
	
	// Use GOPATH version
	return sh.Run(golangciLint, "run", "./...")
}

// ModTidy runs go mod tidy on all modules
func ModTidy() error {
	fmt.Println("📦 Tidying Go modules...")
	
	// Find all go.mod files
	modules, err := findGoModules()
	if err != nil {
		return fmt.Errorf("failed to find Go modules: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, module := range modules {
		wg.Add(1)
		go func(modPath string) {
			defer wg.Done()
			
			dir := filepath.Dir(modPath)
			oldDir, _ := os.Getwd()
			os.Chdir(dir)
			defer os.Chdir(oldDir)
			
			if err := sh.Run("go", "mod", "tidy"); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to tidy %s: %w", modPath, err))
				mu.Unlock()
			} else {
				fmt.Printf("✅ Tidied %s\n", modPath)
			}
		}(module)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("mod tidy failed: %v", errors)
	}

	return nil
}

// CI runs all checks suitable for CI/CD pipeline
func CI() {
	mg.SerialDeps(Clean, ModTidy, Lint, Test, Build)
}

// LambdaFunction represents a Lambda function to build
type LambdaFunction struct {
	Name string
	Path string
}

// findLambdaFunctions discovers all Lambda function directories
func findLambdaFunctions() ([]LambdaFunction, error) {
	var functions []LambdaFunction

	err := filepath.Walk("src", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip node_modules and other non-Go directories
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == ".nx" || info.Name() == ".git") {
			return filepath.SkipDir
		}

		// Look for main.go files in lambda directories
		if info.Name() == "main.go" && strings.Contains(path, "/lambda/") {
			dir := filepath.Dir(path)
			name := filepath.Base(dir)
			functions = append(functions, LambdaFunction{
				Name: name,
				Path: dir,
			})
		}

		return nil
	})

	return functions, err
}

// findGoModules discovers all go.mod files
func findGoModules() ([]string, error) {
	var modules []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "go.mod" {
			modules = append(modules, path)
		}

		return nil
	})

	return modules, err
}

// ensureMutationTester checks if go-mutesting is installed and installs if needed
func ensureMutationTester() error {
	// Check if go-mutesting is available
	if err := sh.Run("go-mutesting", "--help"); err != nil {
		fmt.Println("📦 Installing go-mutesting from Avito-tech...")
		if err := sh.Run("go", "install", "github.com/avito-tech/go-mutesting/cmd/go-mutesting@latest"); err != nil {
			return fmt.Errorf("failed to install go-mutesting: %w", err)
		}
		fmt.Println("✅ go-mutesting installed successfully")
	}
	return nil
}

// findModulesWithTests automatically discovers all Go modules that have test files
func findModulesWithTests() ([]string, error) {
	var modules []string
	
	// Walk the src directory to find all directories with test files
	err := filepath.Walk("src", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip hidden directories and common non-source directories
		if info.IsDir() && (info.Name()[0] == '.' || info.Name() == "node_modules") {
			return filepath.SkipDir
		}
		
		// Check if this directory has test files and Go files
		if info.IsDir() && hasTests(path) && hasGoFiles(path) {
			modules = append(modules, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to walk src directory: %w", err)
	}
	
	fmt.Printf("🔍 Auto-discovered %d modules with tests:\n", len(modules))
	for _, module := range modules {
		fmt.Printf("   - %s\n", module)
	}
	
	return modules, nil
}

// hasTests checks if a directory has Go test files
func hasTests(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	
	matches, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	return err == nil && len(matches) > 0
}

// hasGoFiles checks if a directory has Go source files (non-test)
func hasGoFiles(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(matches) == 0 {
		return false
	}
	
	// Check if there's at least one non-test Go file
	for _, match := range matches {
		if !strings.HasSuffix(match, "_test.go") {
			return true
		}
	}
	
	return false
}

// runMutationTests runs mutation testing on multiple modules in parallel
func runMutationTests(modules []string, outputDir string) error {
	const mutationThreshold = 95.0
	
	// Use worker pool for parallel execution
	maxWorkers := min(runtime.NumCPU(), len(modules))
	workQueue := make(chan string, len(modules))
	results := make(chan MutationResult, len(modules))
	
	// Start workers
	for i := 0; i < maxWorkers; i++ {
		go mutationWorker(i, workQueue, results, outputDir)
	}
	
	// Queue all work
	for _, module := range modules {
		workQueue <- module
	}
	close(workQueue)
	
	// Collect results
	var allResults []MutationResult
	for i := 0; i < len(modules); i++ {
		result := <-results
		allResults = append(allResults, result)
	}
	
	// Generate report and check threshold
	return generateMutationReport(allResults, outputDir, mutationThreshold)
}

// runSingleModuleMutation runs mutation testing on a single module
func runSingleModuleMutation(module, outputDir string) error {
	const mutationThreshold = 95.0
	
	result := runMutationTestOnModule(module, outputDir)
	
	fmt.Printf("🎯 Module: %s\n", result.Module)
	fmt.Printf("   Score: %.1f%%\n", result.Score)
	fmt.Printf("   Status: %s\n", result.Status)
	
	if result.Score < mutationThreshold {
		return fmt.Errorf("mutation score %.1f%% is below threshold %.1f%%", result.Score, mutationThreshold)
	}
	
	return result.Error
}

// MutationResult holds the result of mutation testing for a module
type MutationResult struct {
	Module string
	Score  float64
	Status string
	Error  error
}

// mutationWorker processes mutation testing tasks from the work queue
func mutationWorker(id int, workQueue <-chan string, results chan<- MutationResult, outputDir string) {
	for module := range workQueue {
		start := time.Now()
		result := runMutationTestOnModule(module, outputDir)
		duration := time.Since(start)
		
		fmt.Printf("🔬 Worker %d: %s - %.1f%% (%v)\n", id, result.Module, result.Score, duration)
		results <- result
	}
}

// runMutationTestOnModule runs mutation testing on a specific module
func runMutationTestOnModule(module, baseOutputDir string) MutationResult {
	// Use a safe module name from the full path
	moduleName := strings.ReplaceAll(module, "/", "-")
	if strings.HasPrefix(moduleName, "src-") {
		moduleName = moduleName[4:] // Remove "src-" prefix
	}
	moduleOutputDir := filepath.Join(baseOutputDir, moduleName)
	
	// Clean any existing directory first
	os.RemoveAll(moduleOutputDir)
	if err := os.MkdirAll(moduleOutputDir, 0755); err != nil {
		return MutationResult{
			Module: filepath.Base(module),
			Score:  0,
			Status: "ERROR",
			Error:  fmt.Errorf("failed to create output directory: %w", err),
		}
	}
	
	// Save current directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	// Change to module directory
	if err := os.Chdir(module); err != nil {
		return MutationResult{
			Module: filepath.Base(module),
			Score:  0,
			Status: "ERROR",
			Error:  fmt.Errorf("failed to change to module directory: %w", err),
		}
	}
	
	// Run mutation testing (use absolute path for log file)
	absModuleOutputDir := filepath.Join(originalDir, moduleOutputDir)
	logFile := filepath.Join(absModuleOutputDir, "mutations.log")
	args := []string{
		"--verbose",
		"--exec-timeout=180",
		"--test-recursive",
		".",
	}
	
	// Run mutation testing and capture output
	output, err := sh.Output("go-mutesting", args...)
	if err != nil {
		return MutationResult{
			Module: filepath.Base(module),
			Score:  0,
			Status: "ERROR",
			Error:  fmt.Errorf("mutation testing failed: %w", err),
		}
	}
	
	// Write output to log file
	if err := os.WriteFile(logFile, []byte(output), 0644); err != nil {
		return MutationResult{
			Module: filepath.Base(module),
			Score:  0,
			Status: "ERROR",
			Error:  fmt.Errorf("failed to write log file: %w", err),
		}
	}
	
	// Parse mutation score from output
	score, err := parseMutationScore(logFile)
	if err != nil {
		return MutationResult{
			Module: filepath.Base(module),
			Score:  0,
			Status: "ERROR",
			Error:  fmt.Errorf("failed to parse mutation score: %w", err),
		}
	}
	
	status := "PASSED"
	if score < 95.0 {
		status = "FAILED"
	}
	
	return MutationResult{
		Module: filepath.Base(module), // Keep original simple name for display
		Score:  score,
		Status: status,
		Error:  nil,
	}
}

// parseMutationScore extracts the mutation score from the log file
func parseMutationScore(logFile string) (float64, error) {
	content, err := os.ReadFile(logFile)
	if err != nil {
		return 0, err
	}
	
	// Look for pattern like "The mutation score is X.XXXXX"
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.Contains(line, "The mutation score is") {
			// Extract decimal value and convert to percentage
			parts := strings.Fields(line)
			for i, part := range parts {
				if strings.Contains(part, ".") && i > 0 && part != "is" {
					// Try to parse as float (score is in decimal format)
					if score, err := strconv.ParseFloat(part, 64); err == nil {
						return score * 100, nil // Convert to percentage
					}
				}
			}
		}
	}
	
	return 0, fmt.Errorf("mutation score not found in output")
}

// generateMutationReport creates a comprehensive mutation testing report
func generateMutationReport(results []MutationResult, outputDir string, threshold float64) error {
	reportFile := filepath.Join(outputDir, "mutation-report.md")
	
	var passed, failed, errored int
	var totalScore float64
	
	report := "# Mutation Testing Report\n\n"
	report += fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("Threshold: %.1f%%\n\n", threshold)
	
	report += "## Results Summary\n\n"
	report += "| Module | Score | Status |\n"
	report += "|--------|-------|--------|\n"
	
	for _, result := range results {
		status := result.Status
		if result.Error != nil {
			status = "ERROR"
			errored++
		} else if result.Score >= threshold {
			passed++
		} else {
			failed++
		}
		
		report += fmt.Sprintf("| %s | %.1f%% | %s |\n", result.Module, result.Score, status)
		totalScore += result.Score
	}
	
	avgScore := totalScore / float64(len(results))
	
	report += "\n## Statistics\n\n"
	report += fmt.Sprintf("- Total Modules: %d\n", len(results))
	report += fmt.Sprintf("- Passed: %d\n", passed)
	report += fmt.Sprintf("- Failed: %d\n", failed)
	report += fmt.Sprintf("- Errors: %d\n", errored)
	report += fmt.Sprintf("- Average Score: %.1f%%\n", avgScore)
	
	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}
	
	fmt.Printf("\n📊 Mutation Testing Summary:\n")
	fmt.Printf("   📈 Average Score: %.1f%%\n", avgScore)
	fmt.Printf("   ✅ Passed: %d/%d modules\n", passed, len(results))
	fmt.Printf("   ❌ Failed: %d modules\n", failed)
	fmt.Printf("   🚨 Errors: %d modules\n", errored)
	fmt.Printf("   📄 Report: %s\n", reportFile)
	
	if failed > 0 || errored > 0 {
		return fmt.Errorf("mutation testing failed: %d modules below threshold, %d errors", failed, errored)
	}
	
	return nil
}

// buildLambdaFunction builds a single Lambda function with aggressive optimizations
func buildLambdaFunction(fn LambdaFunction) error {
	env := map[string]string{
		"GOOS":         "linux",
		"GOARCH":       "amd64", // Consider arm64 for 20% cost savings
		"CGO_ENABLED":  "0",
		"GO111MODULE":  "on",
	}

	// Build with aggressive optimizations and build cache
	// Use absolute paths to avoid directory changing issues in concurrent execution
	outputPath := filepath.Join(fn.Path, "main")
	args := []string{
		"build",
		"-ldflags", "-s -w -buildid=", // Remove debug info and build ID for smaller binaries
		"-trimpath",                   // Remove absolute paths for reproducible builds
		"-buildvcs=false",            // Disable VCS info for faster builds
		"-o", outputPath,             // Output to absolute path
		"./" + fn.Path,               // Build the Lambda directory
	}
	
	return sh.RunWith(env, "go", args...)
}