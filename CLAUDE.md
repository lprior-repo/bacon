# Bacon Repository Architecture Guide

## Core Philosophy

This repository follows a **vertical slice architecture** with extreme DRY principles, using **Mage as the single source of truth** for all build, test, and deployment operations. Every architectural decision prioritizes maintainability, testability, and performance through Go's superior concurrency.

## Architecture Principles

### 1. Vertical Slice Architecture
- **All related functionality lives together** - Changes to a feature should only require modifications within a single directory
- **Domain-driven organization** - Each plugin represents a complete 3rd party integration domain
- **Minimal cross-cutting concerns** - Only shared utilities go in the `shared/` folder
- **Self-contained plugins** - Each plugin is completely independent with its own types, clients, parsers, and lambda functions

### 2. Extreme DRY (Don't Repeat Yourself)
- **Extract common patterns** immediately when identified
- **Shared utilities** in `shared/` folder for cross-cutting concerns
- **Template-based code generation** for repetitive structures
- **Configuration-driven behavior** over code duplication

### 3. Pure Functions First
- **Stateless functions** with predictable inputs/outputs
- **Side effects isolated** to clearly defined boundaries
- **Composable design** - functions can be easily combined and tested
- **Immutable data structures** where possible

### 4. AWS Serverless Architecture
- **Lambda Functions**: Single responsibility, pure functional design
- **Step Functions**: Orchestrate complex workflows without local concurrency
- **EventBridge/CloudWatch**: Event-driven triggers for data processing
- **No Local Concurrency**: AWS handles all scaling and parallel execution

### 5. Functional Programming with samber/lo
- **Mandatory samber/lo Usage**: All data transformations use lo.Map, lo.Filter, lo.Reduce
- **Zero Imperative Loops**: No for/while loops in business logic
- **Pure Transformation Functions**: All data processing functions are side-effect free
- **Native Go Error Handling**: Standard Go error propagation patterns

## Directory Structure

```
bacon/
├── src/                              # Application source code (DDD slices)
│   ├── plugins/                      # 3rd party integrations (completely self-contained)
│   │   ├── github/                   # GitHub integration domain
│   │   │   ├── types/               # GitHub-specific types
│   │   │   ├── parsers/             # Code parsing logic
│   │   │   ├── clients/             # GitHub API clients
│   │   │   ├── cache/               # GitHub-specific caching
│   │   │   └── lambda/              # GitHub Lambda functions
│   │   │       ├── github-scraper/
│   │   │       └── codeowners-scraper/
│   │   ├── datadog/                 # Datadog integration domain
│   │   │   ├── types/               # Datadog-specific types
│   │   │   ├── clients/             # Datadog API clients  
│   │   │   ├── shared/              # Shared functional utilities
│   │   │   └── lambda/              # Datadog Lambda functions (AWS Serverless)
│   │   │       ├── datadog-teams-scraper/     # Teams API v2 scraper
│   │   │       ├── datadog-users-scraper/     # Users API v2 scraper
│   │   │       ├── datadog-services-scraper/  # Service Catalog API v2 scraper
│   │   │       ├── datadog-organizations-scraper/ # Organizations API v2 scraper
│   │   │       └── datadog-orchestrator/      # Step Functions orchestrator
│   │   └── openshift/               # OpenShift integration domain
│   │       ├── types/               # OpenShift-specific types
│   │       ├── clients/             # OpenShift API clients
│   │       └── lambda/              # OpenShift Lambda functions
│   │           └── openshift-scraper/
│   └── shared/                       # Cross-cutting concerns and internal services
│       ├── api/                     # Internal GraphQL API
│       │   ├── resolvers/
│       │   │   ├── mutation/
│       │   │   └── query/
│       │   └── schemas/
│       ├── relationship-finding/     # Internal data processing
│       └── [utilities]              # Shared utilities (aws.go, functional.go, etc.)
├── terraform/                       # Infrastructure as Code
├── magefile.go                      # **SINGLE SOURCE OF TRUTH** for all build/test/deploy commands
├── tests/                           # Testing suites
│   ├── unit/
│   ├── integration/
│   ├── contract/
│   └── e2e/
├── go.mod                           # Single consolidated module
└── nx.json                          # NX workspace configuration
```

## Build System - Mage as Single Source of Truth

**CRITICAL**: All build, test, and deployment operations are defined in `magefile.go`. No shell scripts or external build files exist.

## Testing Strategy

### Comprehensive Testing Pyramid

1. **Unit Tests** (`*_test.go`)
   - Test individual functions in isolation
   - Fast execution (< 1ms per test)
   - High coverage (aim for 100%)

2. **Property-Based Tests** (using `testing/quick` or `gopter`)
   - Generate random inputs to test invariants
   - Discover edge cases automatically
   - Test properties that should always hold

3. **Fuzz Tests** (`*_fuzz_test.go`)
   - Generate random byte sequences
   - Test parsing and input validation
   - Discover security vulnerabilities

4. **Mutation Tests** (using `go-mutesting`)
   - Verify test quality by introducing bugs
   - **95% threshold** - tests must catch 95% of mutations
   - Automated via mage: `mage testMutation`

5. **Integration Tests** (`tests/integration/`)
   - Test component interactions
   - Use real dependencies where possible
   - Test error scenarios and edge cases

6. **Contract Tests** (`tests/contract/`)
   - Verify Terraform configurations
   - Test infrastructure contracts
   - Validate resource configurations

7. **End-to-End Tests** (`tests/e2e/`)
   - Test complete user journeys
   - Use ephemeral infrastructure
   - Validate business requirements

### All Commands via Mage

**NEVER use bash scripts or NX commands directly**. All operations go through Mage:

```bash
# TESTING COMMANDS
mage test                 # All tests with race detection and coverage
mage testUnit             # Unit tests only (excludes integration tests)
mage testContract         # Contract tests against Terraform
mage testE2E              # End-to-end tests against ephemeral infrastructure
mage testMutation         # Mutation testing (95% threshold)
mage testMutationModule   # Mutation test specific module

# BUILD COMMANDS  
mage build                # Build all Lambda functions in parallel
mage clean                # Remove all build artifacts and cache

# QUALITY COMMANDS
mage lint                 # Run golangci-lint on all code
mage modTidy              # Run go mod tidy on the consolidated module

# CI/CD COMMANDS
mage ci                   # Complete CI pipeline (clean, modTidy, lint, test, build)
```

## Build System - Mage Only

### Core Principles
- **Single Source of Truth**: `magefile.go` contains ALL build logic
- **No Shell Scripts**: Zero dependency on bash/shell scripts
- **Go Concurrency**: Leverages Go's superior parallelization for builds and tests
- **Intelligent Discovery**: Automatically finds Lambda functions in plugin structure

### Build Optimizations

- **Parallel builds** using worker pools
- **Aggressive compiler flags** (`-ldflags "-s -w"` for smaller binaries)
- **Build cache utilization** for incremental builds
- **Static linking** (`CGO_ENABLED=0`) for portable binaries

## Development Workflow

### 1. Plugin Development (3rd Party Integrations)
```bash
# Create new feature branch
git checkout -b feature/github-enhancement

# Work in the appropriate plugin directory
# src/plugins/github/ - for GitHub-related changes
# src/plugins/datadog/ - for Datadog-related changes  
# src/plugins/openshift/ - for OpenShift-related changes

# Tidy dependencies and build
mage modTidy
mage build

# Run tests locally
mage test
```

### 2. Shared Component Development
```bash
# Work in shared directory for internal services
# src/shared/api/ - for GraphQL API changes
# src/shared/relationship-finding/ - for data processing
# src/shared/ - for utility functions

# Full development cycle through Mage
mage clean
mage modTidy  
mage lint
mage test
mage build
```

### 3. Quality Assurance (Required Before Commit)
```bash
# Complete CI pipeline locally
mage ci

# Or run individual steps
mage clean
mage modTidy
mage lint  
mage test
mage testMutation    # Ensure 95% mutation coverage
mage build
```

### 4. Infrastructure Changes
```bash
# Navigate to terraform directory
cd terraform/

# Use terraform commands directly (not through Mage)
terraform plan
terraform apply
```

## Module Organization

### Single Module Strategy
- **One consolidated go.mod** at repository root for all packages
- **Consistent dependency versions** across all plugins and shared components
- **No module replace directives** - everything references the single module
- **Simplified builds** - `mage build` works across entire codebase

### Plugin Dependencies
```
src/shared/ (foundational utilities and internal services)
    ↑
src/plugins/github/ (self-contained GitHub integration)
src/plugins/datadog/ (self-contained Datadog integration)  
src/plugins/openshift/ (self-contained OpenShift integration)

Each plugin can import from src/shared/ but NOT from other plugins
```

## Adding New Plugins

### Manual Plugin Creation
Since we prioritize simplicity, create new plugins manually following the established pattern:

```bash
# Create new plugin structure
mkdir -p src/plugins/newservice/{types,clients,lambda}

# Copy existing plugin as template
cp -r src/plugins/github/lambda/github-scraper src/plugins/newservice/lambda/newservice-scraper

# Update import paths and implement new service logic
# All operations through Mage will automatically discover the new plugin
```

## Performance Guidelines

### Concurrency Patterns
- **Worker pools** for parallel processing
- **Channel-based communication** over shared memory
- **Context-based cancellation** for graceful shutdowns
- **Semaphores** for resource limiting

### Memory Management
- **Streaming processing** for large datasets
- **Pool objects** for frequent allocations
- **Explicit garbage collection** tuning where needed

### Lambda Optimization
- **Cold start reduction** through smaller binaries
- **Connection pooling** for database access
- **Lazy initialization** of expensive resources

## Deployment Strategy

### Environment Management
- **Namespace-based deployments** via Terraform variables
- **Branch-based environments** for feature development
- **Production deployment** only from main branch

### Infrastructure as Code
- **All infrastructure** defined in `terraform/` directory
- **Module-based organization** for reusability
- **State management** via remote backends
- **Security scanning** via Checkov integration

## Troubleshooting

### Common Issues

1. **Build Failures**
   ```bash
   # Always use Mage - never direct go commands
   mage clean
   mage modTidy
   mage build
   ```

2. **Test Failures**
   ```bash
   # Use Mage for all testing
   mage test                # Run all tests
   mage testUnit           # Run only unit tests
   mage testMutation      # Check mutation coverage
   ```

3. **Quality Issues**
   ```bash
   # Full quality pipeline
   mage ci                 # Complete CI pipeline
   mage lint              # Just linting
   ```

4. **Terraform Issues**
   ```bash
   # Navigate to terraform directory first
   cd terraform/
   
   # Use terraform directly (not through Mage)
   terraform refresh
   TF_LOG=DEBUG terraform plan
   ```

### Performance Debugging
- **Go profiling** tools for CPU and memory analysis
- **Distributed tracing** for Lambda function analysis
- **CloudWatch metrics** for production monitoring

## Best Practices

### Code Style
- Follow `gofmt` and `golangci-lint` standards
- Use descriptive variable names
- Write self-documenting code
- Add comments for complex business logic

### Testing Best Practices
- Write tests before implementation (TDD)
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error conditions explicitly

### Security Considerations
- Never commit secrets or credentials
- Use AWS IAM roles and policies appropriately
- Validate all inputs and sanitize outputs
- Implement proper error handling without information leakage

### Documentation
- Update this CLAUDE.md for architectural changes
- Document complex algorithms inline
- Maintain README files for each major component
- Use godoc conventions for public APIs

## Continuous Integration

### GitHub Actions Workflow
All CI operations use Mage commands:

```yaml
# Example GitHub Actions step
- name: Build and Test
  run: |
    mage ci              # Complete CI pipeline
    mage testMutation   # Ensure 95% mutation coverage
```

### Quality Gates (All via Mage)
- `mage test` must pass with 95% mutation coverage
- `mage lint` must pass without warnings  
- `mage build` must successfully build all Lambda functions
- Terraform plan must validate successfully (direct terraform commands)

## Summary

This repository uses **Mage as the single source of truth** for all build, test, and deployment operations. The DDD plugin architecture ensures clear boundaries between 3rd party integrations while shared components provide common functionality.

### Key Commands to Remember
- `mage ci` - Complete CI pipeline
- `mage build` - Build all Lambda functions  
- `mage test` - Run all tests with coverage
- `mage testMutation` - Ensure 95% mutation coverage
- `mage lint` - Code quality checks
- `mage clean` - Clean all artifacts

**Never use shell scripts, NX commands directly, or manual go commands. Always use Mage.**

## Datadog Plugin AWS Serverless Architecture

### Lambda Functions (Single Responsibility)
Each Datadog Lambda function scrapes one specific Datadog API v2 endpoint using pure functional programming:

- **datadog-teams-scraper/** - Teams API v2 endpoint scraping
- **datadog-users-scraper/** - Users API v2 endpoint scraping  
- **datadog-services-scraper/** - Service Catalog API v2 endpoint scraping
- **datadog-organizations-scraper/** - Organizations API v2 endpoint scraping
- **datadog-orchestrator/** - Step Function orchestration Lambda

### Functional Programming Requirements
- **samber/lo Mandatory**: All data transformations must use lo.Map, lo.Filter, lo.Reduce
- **No Imperative Loops**: Zero for/while loops in business logic
- **Pure Functions Only**: No side effects in transformation logic
- **Immutable Data**: All data structures immutable with functional updates
- **Native Go Errors**: Standard error propagation patterns
- **Single Responsibility**: Each Lambda handles one API endpoint

### Architecture Pattern
```
EventBridge/CloudWatch → Orchestrator Lambda → Step Functions → Parallel Lambda Execution
                                                              ↓
                                     Teams/Users/Services/Orgs Scrapers → DynamoDB
```

### Datadog API v2 Integration
- **Teams API**: Complete team structure, membership, and relationships
- **Users API**: User-team associations, roles, and organizational structure
- **Service Catalog API**: Service-team ownership mappings and metadata
- **Organizations API**: Organizational hierarchy and configuration

### Data Flow Architecture
1. **Trigger**: EventBridge or CloudWatch event triggers orchestrator
2. **Orchestration**: Step Functions coordinate parallel Lambda execution
3. **Processing**: Each Lambda uses pure functions with samber/lo transformations
4. **Storage**: Results stored in DynamoDB with proper indexing
5. **Relationships**: Team-user-service mappings built functionally

All functions follow the established functional programming patterns with samber/lo and maintain pure, composable design.