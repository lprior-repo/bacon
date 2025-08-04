# Build System with Mage + NX

This project uses a hybrid build system combining [Mage](https://magefile.org/) for Go Lambda functions and [NX](https://nx.dev/) for workspace management with the new DDD plugin architecture.

## Quick Start

1. **Install dependencies**:
   ```bash
   # Install Node.js dependencies (NX)
   npm install
   
   # Install Mage for Go builds
   go install github.com/magefile/mage@latest
   ```

2. **Build all Lambda functions** (using Mage):
   ```bash
   mage build
   ```

3. **Build all projects** (using NX):
   ```bash
   npm run build
   # or directly: nx run-many -t build
   ```

4. **Run full CI pipeline**:
   ```bash
   mage ci
   ```

## Available Commands

### Mage Commands (Go-specific)
- `mage build` - Build all Lambda functions in parallel
- `mage clean` - Remove build artifacts and cache
- `mage modtidy` - Run `go mod tidy` on the consolidated module
- `mage test` - Run tests with race detection in parallel
- `mage testmutation` - Run mutation testing (95%+ coverage required)
- `mage lint` - Run golangci-lint on all code
- `mage ci` - Run complete CI pipeline (clean, modtidy, lint, test, build)

### NX Commands (Workspace-wide)
- `npm run build` / `nx run-many -t build` - Build all projects using NX
- `npm run test` / `nx run-many -t test` - Test all projects
- `npm run lint` / `nx run-many -t lint` - Lint all projects
- `npm run clean` / `nx run-many -t clean` - Clean all projects

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

### Mage Discovery (Go Lambda Functions)
Mage automatically discovers Lambda functions in the new DDD plugin architecture:
- **Plugin Lambda Functions**: `src/plugins/*/lambda/*/main.go`
- **Shared Lambda Functions**: `src/shared/*/lambda/*/main.go` (if any)
- **Go Modules**: Single consolidated `go.mod` at the root
- **Build Targets**: Dynamically builds based on project structure

### NX Discovery (All Projects)
NX automatically discovers projects based on:
- **Project Configuration**: All `project.json` files in plugin and shared directories
- **Affected Analysis**: Only builds/tests changed projects and their dependencies
- **Workspace Structure**: Understands the relationship between plugins and shared components

## Example Output

### Mage Build Output
```bash
$ mage build
Building 5 Lambda functions with optimized parallelism...
✅ Worker 0: Built github-scraper (1.2s)
✅ Worker 1: Built codeowners-scraper (1.1s)
✅ Worker 2: Built datadog-scraper (1.3s)
✅ Worker 0: Built openshift-scraper (1.4s)
✅ Worker 3: Built query (1.0s)
🎉 Successfully built all 5 Lambda functions
```

### NX Build Output
```bash
$ npm run build

> bacon@1.0.0 build
> nx run-many -t build

✓  nx run github:build (2.1s)
✓  nx run datadog:build (1.8s)
✓  nx run openshift:build (1.9s)
✓  nx run shared:build (1.2s)

 ————————————————————————————————————————————————————

 >  NX   Successfully ran target build for 4 projects (2.1s)

    Nx read the output from the cache instead of running the command for 1 out of 4 tasks.
```

## GitHub Actions Usage

The CI/CD pipeline leverages both Mage and NX for optimal performance:

### Build Strategy
- **NX Affected Builds**: Only builds changed plugins and their dependencies
- **Mage Lambda Builds**: Parallel compilation of all Lambda functions
- **Intelligent Caching**: NX caches build artifacts across pipeline runs
- **Terraform Deployment**: Uses namespace-based deployments for environment isolation

### Pipeline Configuration
```yaml
# Build stage - uses both systems
- name: Build affected projects (NX)
  run: npx nx affected -t build

- name: Build Lambda functions (Mage)  
  run: mage build

# Deploy stage
- name: Deploy infrastructure
  run: terraform apply -var="namespace=${{ github.ref_name }}"
```

### Environment Isolation
- **Branch deployments**: `terraform apply -var="namespace=$BRANCH_NAME"`
- **Production deployment**: `terraform apply -var="namespace=prod"`
- **Plugin isolation**: Each plugin can be deployed independently

This hybrid approach ensures:
- **Fastest possible builds**: Only affected code is rebuilt
- **Consistent Lambda artifacts**: Mage ensures optimized Go binaries
- **Environment safety**: Namespace isolation prevents conflicts
- **Cache efficiency**: NX provides intelligent caching across runs