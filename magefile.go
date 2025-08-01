//go:build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

// Build compiles all Lambda functions in parallel for maximum speed
func Build() error {
	mg.Deps(Clean)

	lambdaFunctions, err := findLambdaFunctions()
	if err != nil {
		return fmt.Errorf("failed to find Lambda functions: %w", err)
	}

	if len(lambdaFunctions) == 0 {
		fmt.Println("No Lambda functions found to build")
		return nil
	}

	fmt.Printf("Building %d Lambda functions in parallel...\n", len(lambdaFunctions))

	// Use all available CPU cores for parallel builds
	maxConcurrency := runtime.NumCPU()
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, function := range lambdaFunctions {
		wg.Add(1)
		go func(fn LambdaFunction) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			if err := buildLambdaFunction(fn); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to build %s: %w", fn.Name, err))
				mu.Unlock()
			} else {
				fmt.Printf("✅ Built %s\n", fn.Name)
			}
		}(function)
	}

	wg.Wait()

	if len(errors) > 0 {
		fmt.Printf("❌ %d builds failed:\n", len(errors))
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("build failed with %d errors", len(errors))
	}

	fmt.Printf("🎉 Successfully built all %d Lambda functions\n", len(lambdaFunctions))
	return nil
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

// Lint runs linters on all Go code
func Lint() error {
	fmt.Println("🔍 Running linters...")
	
	// Check if golangci-lint is available
	if err := sh.Run("golangci-lint", "--version"); err != nil {
		fmt.Println("golangci-lint not found, installing...")
		if err := sh.Run("go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"); err != nil {
			return fmt.Errorf("failed to install golangci-lint: %w", err)
		}
	}
	
	return sh.Run("golangci-lint", "run", "./...")
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

// buildLambdaFunction builds a single Lambda function
func buildLambdaFunction(fn LambdaFunction) error {
	env := map[string]string{
		"GOOS":         "linux",
		"GOARCH":       "amd64",
		"CGO_ENABLED":  "0",
		"GO111MODULE":  "on",
	}

	// Build the Lambda function
	outputPath := filepath.Join(fn.Path, "main")
	return sh.RunWith(env, "go", "build", "-ldflags", "-s -w", "-o", outputPath, "./"+fn.Path)
}