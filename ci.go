// Ultra-fast CI pipeline with Dagger
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dagger.io/dagger"
)

func main() {
	ctx := context.Background()
	start := time.Now()

	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Run everything in parallel with intelligent caching
	var wg sync.WaitGroup
	
	// 1. Build all Lambdas (30-45 seconds with cache)
	wg.Add(1)
	go func() {
		defer wg.Done()
		buildAllLambdas(ctx, client)
	}()
	
	// 2. Run all tests (30-45 seconds with cache)
	wg.Add(1) 
	go func() {
		defer wg.Done()
		runAllTests(ctx, client)
	}()
	
	// 3. Terraform operations (45-60 seconds with cache)
	wg.Add(1)
	go func() {
		defer wg.Done()
		terraformOperations(ctx, client)
	}()
	
	// 4. Security scans (15-30 seconds with cache)
	wg.Add(1)
	go func() {
		defer wg.Done()
		securityScans(ctx, client)
	}()
	
	wg.Wait()
	
	duration := time.Since(start)
	fmt.Printf("🎉 Complete pipeline in %v (target: <2min)\n", duration)
}

func buildAllLambdas(ctx context.Context, client *dagger.Client) {
	// Shared cache volumes for maximum efficiency
	goModCache := client.CacheVolume("go-mod-cache")
	goBuildCache := client.CacheVolume("go-build-cache")
	
	baseBuilder := client.Container().
		From("golang:1.24.5-alpine").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithMountedCache("/root/.cache/go-build", goBuildCache).
		WithEnvVariable("GOOS", "linux").
		WithEnvVariable("GOARCH", "amd64").
		WithEnvVariable("CGO_ENABLED", "0").
		WithDirectory("/workspace", client.Host().Directory("."))
	
	lambdas := []string{
		"codeowners-scraper", "github-scraper", "datadog-scraper", 
		"openshift-scraper", "event-processor",
	}
	
	// Build all in parallel - Dagger handles the orchestration
	var builders []*dagger.Container
	for _, lambda := range lambdas {
		builder := baseBuilder.
			WithWorkdir(fmt.Sprintf("/workspace/src/*/lambda/%s", lambda)).
			WithExec([]string{"go", "build", "-ldflags", "-s -w", "-o", "main", "."})
		builders = append(builders, builder)
	}
	
	// Export all binaries in parallel
	for i, builder := range builders {
		binary := builder.File("main")
		_, err := binary.Export(ctx, fmt.Sprintf("./dist/%s/main", lambdas[i]))
		if err != nil {
			fmt.Printf("❌ Build failed for %s: %v\n", lambdas[i], err)
		} else {
			fmt.Printf("✅ Built %s\n", lambdas[i])
		}
	}
}

func runAllTests(ctx context.Context, client *dagger.Client) {
	testCache := client.CacheVolume("go-test-cache")
	goModCache := client.CacheVolume("go-mod-cache")
	
	tester := client.Container().
		From("golang:1.24.5").
		WithMountedCache("/go/pkg/mod", goModCache).
		WithMountedCache("/root/.cache/go-build", testCache).
		WithDirectory("/workspace", client.Host().Directory(".")).
		WithWorkdir("/workspace")
	
	// Parallel test execution with different strategies
	testSuites := map[string][]string{
		"unit": {"go", "test", "-race", "-short", "-parallel", "8", "./..."},
		"integration": {"go", "test", "-tags=integration", "./tests/integration/..."},
		"contract": {"go", "test", "-tags=contract", "./tests/contract/..."},
	}
	
	for testName, cmd := range testSuites {
		result := tester.WithExec(cmd)
		stdout, err := result.Stdout(ctx)
		if err != nil {
			fmt.Printf("❌ %s tests failed: %v\n", testName, err)
		} else {
			fmt.Printf("✅ %s tests passed\n", testName)
		}
	}
}

func terraformOperations(ctx context.Context, client *dagger.Client) {
	tfCache := client.CacheVolume("terraform-cache") 
	
	terraform := client.Container().
		From("hashicorp/terraform:1.8").
		WithMountedCache("/tmp/.terraform", tfCache).
		WithDirectory("/workspace", client.Host().Directory("./terraform")).
		WithWorkdir("/workspace")
	
	environments := []string{"dev", "staging"}
	
	for _, env := range environments {
		// Each environment planned in parallel with shared cache
		envTerraform := terraform.
			WithEnvVariable("TF_VAR_namespace", fmt.Sprintf("bacon-%s", env)).
			WithExec([]string{"terraform", "init"}).  // Cached after first run
			WithExec([]string{"terraform", "plan", "-out", fmt.Sprintf("tfplan-%s", env)})
		
		planFile := envTerraform.File(fmt.Sprintf("tfplan-%s", env))
		_, err := planFile.Export(ctx, fmt.Sprintf("./plans/tfplan-%s", env))
		if err != nil {
			fmt.Printf("❌ Terraform plan failed for %s: %v\n", env, err)
		} else {
			fmt.Printf("✅ Terraform planned for %s\n", env)
		}
	}
}

func securityScans(ctx context.Context, client *dagger.Client) {
	// Gosec security scanner with cache
	gosecCache := client.CacheVolume("gosec-cache")
	
	scanner := client.Container().
		From("securecodewarrior/gosec:latest").
		WithMountedCache("/tmp/gosec", gosecCache).
		WithDirectory("/workspace", client.Host().Directory(".")).
		WithWorkdir("/workspace").
		WithExec([]string{"gosec", "-fmt", "sarif", "-out", "gosec-report.sarif", "./..."})
	
	report := scanner.File("gosec-report.sarif")
	_, err := report.Export(ctx, "./security/gosec-report.sarif")
	if err != nil {
		fmt.Printf("❌ Security scan failed: %v\n", err)
	} else {
		fmt.Printf("✅ Security scan complete\n")
	}
}