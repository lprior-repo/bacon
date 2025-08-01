# Build System with Mage

This project uses [Mage](https://magefile.org/) for fast, parallel builds of Go Lambda functions.

## Quick Start

1. **Install Mage**:
   ```bash
   go install github.com/magefile/mage@latest
   ```

2. **Build all Lambda functions**:
   ```bash
   mage build
   ```

3. **Run full CI pipeline**:
   ```bash
   mage ci
   ```

## Available Commands

### Build Commands
- `mage build` - Build all Lambda functions in parallel
- `mage clean` - Remove build artifacts and cache
- `mage modtidy` - Run `go mod tidy` on all modules in parallel

### Quality Commands
- `mage test` - Run tests with race detection in parallel
- `mage lint` - Run golangci-lint on all code
- `mage ci` - Run complete CI pipeline (clean, modtidy, lint, test, build)

## Performance Benefits

### Parallel Execution
- **Lambda Builds**: All Lambda functions build simultaneously using all CPU cores
- **Module Management**: `go mod tidy` runs in parallel across all modules
- **Testing**: Tests run with parallel execution enabled

### Build Optimization
- **Compiler Flags**: Uses `-ldflags "-s -w"` to reduce binary size
- **Static Linking**: `CGO_ENABLED=0` for portable Linux binaries
- **Build Cache**: Leverages Go build cache for incremental builds

### GitHub Actions Integration
The CI pipeline automatically:
1. Builds all Lambda functions in parallel
2. Runs linting and tests concurrently
3. Uploads build artifacts
4. Runs Terraform plan with dynamic namespace from branch name
5. Deploys to production on main branch

## Architecture Discovery

Mage automatically discovers:
- **Lambda Functions**: Finds all `main.go` files in `src/*/lambda/*` directories
- **Go Modules**: Discovers all `go.mod` files for parallel tidying
- **Build Targets**: Dynamically builds based on project structure

## Example Output

```bash
$ mage build
Building 6 Lambda functions in parallel...
✅ Built codeowners-scraper
✅ Built github-scraper
✅ Built datadog-scraper
✅ Built openshift-scraper
✅ Built event-processor
✅ Built mutation
✅ Built query
🎉 Successfully built all 7 Lambda functions
```

## GitHub Actions Usage

The workflow uses namespace-based deployments:
- **Branch deployments**: `terraform apply -var="namespace=$BRANCH_NAME"`
- **Production deployment**: `terraform apply -var="namespace=prod"`
- **Parallel builds**: Utilizes all runner cores for maximum speed

This approach ensures:
- Fast builds on CI/CD runners
- Environment isolation through namespaces
- Consistent builds across local and CI environments