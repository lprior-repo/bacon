# Bacon: AWS Serverless Infrastructure Discovery Platform

**A Domain-Driven Design (DDD) microservices platform for comprehensive AWS infrastructure discovery and analysis**

## Overview

Bacon is a serverless AWS infrastructure discovery and analysis platform designed to provide comprehensive visibility into multi-cloud environments. The system aggregates data from GitHub repositories, DataDog metrics, OpenShift clusters, and AWS resources to create a unified ownership and relationship graph stored in Amazon Neptune.

The platform follows Domain-Driven Design (DDD) principles with a plugin-based architecture, where each integration is completely self-contained and independent.

## Architecture

### DDD Plugin-Based Architecture

Bacon uses a **Domain-Driven Design (DDD)** approach with a plugin-based architecture that provides:

- **Plugin Isolation**: Each third-party integration is completely self-contained
- **Shared Components**: Common functionality is centralized in shared modules
- **Single Build System**: Consolidated nx.json + Mage for unified builds
- **Single Go Module**: All modules consolidated into one go.mod file
- **Clear Boundaries**: Each plugin represents a bounded context

### Core Components

- **Compute**: AWS Lambda functions for serverless processing
- **Orchestration**: AWS Step Functions for workflow management  
- **Storage**: Amazon S3 for data storage, DynamoDB for state management
- **Graph Database**: Amazon Neptune for relationship storage
- **API**: GraphQL API via AWS AppSync for flexible querying
- **Networking**: VPC with private subnets for secure deployment
- **Monitoring**: CloudWatch for logging and monitoring

## Project Structure

```
bacon/
├── src/
│   ├── plugins/                    # 3rd party integrations (DDD bounded contexts)
│   │   ├── github/                # GitHub integration plugin
│   │   │   ├── clients/           # GitHub API clients
│   │   │   ├── lambda/            # GitHub Lambda functions
│   │   │   │   ├── github-scraper/
│   │   │   │   └── codeowners-scraper/
│   │   │   ├── parsers/           # GitHub-specific parsers
│   │   │   ├── types/             # GitHub domain types
│   │   │   └── project.json       # NX project configuration
│   │   ├── datadog/               # DataDog integration plugin
│   │   │   ├── clients/           # DataDog API clients
│   │   │   ├── lambda/            # DataDog Lambda functions
│   │   │   │   └── datadog-scraper/
│   │   │   ├── types/             # DataDog domain types
│   │   │   └── project.json       # NX project configuration
│   │   └── openshift/             # OpenShift integration plugin
│   │       ├── clients/           # OpenShift/K8s API clients
│   │       ├── lambda/            # OpenShift Lambda functions
│   │       │   └── openshift-scraper/
│   │       ├── types/             # OpenShift domain types
│   │       └── project.json       # NX project configuration
│   └── shared/                    # Shared components across plugins
│       ├── api/                   # GraphQL API and resolvers
│       │   ├── resolvers/         # GraphQL resolvers
│       │   │   ├── query/         # Query resolvers
│       │   │   └── mutation/      # Mutation resolvers
│       │   └── schemas/           # GraphQL schemas
│       ├── relationship-finding/   # Core relationship discovery logic
│       ├── aws.go                 # AWS SDK utilities
│       ├── functional.go          # Functional programming utilities
│       ├── response.go            # HTTP response utilities
│       ├── testing.go             # Test utilities
│       ├── tracing.go             # X-Ray tracing utilities
│       └── project.json           # NX project configuration
├── terraform/                     # Infrastructure as Code
├── tests/                         # Test suites
│   ├── contract/                  # Contract tests
│   ├── e2e/                       # End-to-end tests
│   └── unit/                      # Unit tests
├── go.mod                         # Single Go module for entire project
├── nx.json                        # NX workspace configuration
├── magefile.go                    # Mage build system
└── package.json                   # Node.js dependencies and scripts
```

## DDD Principles Applied

### 1. Bounded Contexts
Each plugin represents a bounded context with:
- **Clear domain boundaries**: GitHub, DataDog, and OpenShift are separate domains
- **Independent models**: Each plugin has its own types and business logic
- **Autonomous evolution**: Plugins can evolve independently

### 2. Plugin Self-Containment
Each plugin includes:
- **Domain-specific clients**: API clients for the external service
- **Business logic**: Lambda functions implementing domain operations  
- **Types**: Domain-specific data structures
- **Tests**: Comprehensive test coverage including property-based tests

### 3. Shared Kernel
The `src/shared/` directory contains:
- **Common utilities**: AWS SDK helpers, tracing, testing utilities
- **API layer**: GraphQL API serving all plugins
- **Relationship finding**: Core cross-domain relationship discovery

## Build System

### Mage + NX Hybrid Approach

The project uses a hybrid build system combining:
- **Mage**: Fast, parallel builds of Go Lambda functions
- **NX**: Intelligent caching and affected builds for the entire workspace

#### Available Commands

**Mage Commands** (Go-specific):
```bash
# Build all Lambda functions in parallel
mage build

# Run full CI pipeline  
mage ci

# Clean build artifacts
mage clean

# Run tests with mutation testing (95%+ coverage required)
mage test
mage testmutation

# Run linting
mage lint

# Tidy Go modules
mage modtidy
```

**NPM/NX Commands** (Workspace-wide):
```bash
# Build all projects using NX
npm run build

# Test all projects  
npm run test

# Lint all projects
npm run lint

# Clean all projects
npm run clean
```

### Automatic Discovery

The build system automatically discovers:

**Lambda Functions**: 
- `src/plugins/*/lambda/*/main.go`
- `src/shared/*/lambda/*/main.go` (if any)

**Go Modules**: All `go.mod` files (consolidated to root)

**NX Projects**: All `project.json` files in plugin and shared directories

## Development Workflow

### 1. Plugin Development
When adding a new integration:

1. **Create plugin directory**: `src/plugins/{service-name}/`
2. **Add domain components**:
   - `clients/` - API client for external service
   - `lambda/` - Lambda function implementations
   - `types/` - Domain-specific types
   - `project.json` - NX project configuration
3. **Implement Lambda functions** with proper error handling and tracing
4. **Add comprehensive tests** including property-based testing
5. **Update terraform** to deploy new Lambda functions

### 2. Shared Component Development
When modifying shared components:

1. **Consider impact** across all plugins
2. **Maintain backward compatibility** where possible
3. **Update all affected tests**
4. **Run full mutation testing** to ensure coverage

### 3. Testing Strategy

**Unit Tests**: Each plugin and shared component has comprehensive unit tests

**Property-Based Tests**: Using `pgregory.net/rapid` for robust testing

**Mutation Tests**: 95%+ mutation testing coverage requirement

**Contract Tests**: API contract testing for external integrations

**E2E Tests**: Full pipeline testing with ephemeral infrastructure

## Prerequisites

### Required Tools
1. **Go**: Version 1.24.5 or later
2. **Node.js**: Version 20 or later  
3. **Mage**: `go install github.com/magefile/mage@latest`
4. **Terraform**: Version 1.0 or later
5. **AWS CLI**: Configured with appropriate credentials

### AWS Infrastructure
- VPC with private subnets
- At least 2 private subnets in different AZs for high availability
- NAT Gateway for Lambda internet access
- IAM roles and policies for Lambda execution

## Configuration

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `AWS_REGION` | AWS region | `us-east-1` |
| `ENV` | Environment name | `dev`/`staging`/`prod` |
| `PROJECT_NAME` | Project name | `bacon` |
| `VPC_ID` | Existing VPC ID | `vpc-0123456789abcdef0` |
| `PRIVATE_SUBNET_IDS` | Private subnet IDs | `subnet-0123abc,subnet-4567def` |

### Setup Steps

1. **Clone repository**:
   ```bash
   git clone <repository-url>
   cd bacon
   ```

2. **Install dependencies**:
   ```bash
   npm install
   go mod download
   ```

3. **Configure Terraform variables**:
   ```bash
   cp terraform/terraform.tfvars.example terraform/terraform.tfvars
   # Edit terraform.tfvars with your values
   ```

4. **Build and test**:
   ```bash
   # Build all Lambda functions
   mage build
   
   # Run all tests
   mage test
   
   # Run mutation tests (95%+ coverage required)
   mage testmutation
   ```

5. **Deploy infrastructure**:
   ```bash
   cd terraform
   terraform init
   terraform plan
   terraform apply
   ```

## API Usage

### GraphQL Endpoint

The platform provides a GraphQL API for querying infrastructure data:

```graphql
query GetResourcesByOwner($teamId: ID!) {
  resourcesByOwner(teamId: $teamId) {
    id
    type
    name
    owners {
      name
      members
    }
    relationships {
      type
      target {
        id
        name
      }
    }
    lastUpdated
  }
}
```

### Common Queries

**Find all resources owned by a team**:
```graphql
query ResourcesByTeam($teamId: ID!) {
  resourcesByOwner(teamId: $teamId) {
    id
    name
    type
    tags
  }
}
```

**Discover relationship paths**:
```graphql  
query RelationshipPath($from: ID!, $to: ID!) {
  relationshipPath(from: $from, to: $to) {
    id
    name
    type
  }
}
```

## Monitoring & Observability

### Multi-Layer Monitoring

1. **Infrastructure**: CloudWatch dashboards for Lambda, Neptune, EventBridge
2. **Application**: X-Ray distributed tracing, custom metrics, structured logging
3. **Business**: Data freshness, resource discovery coverage, API usage patterns

### Key Metrics

| Metric | Target | Description |
|--------|--------|-------------|
| API Response Time | <200ms (p95) | GraphQL query performance |
| Resource Discovery Coverage | >95% | Percentage of resources discovered |
| Data Freshness | <5 minutes | Time from source update to availability |
| System Availability | 99.9% | Overall platform availability |
| Mutation Test Coverage | 95%+ | Code quality assurance |

## Security

### Multi-Layer Security

1. **Network**: VPC isolation, private subnets, security groups
2. **Identity**: IAM roles with least privilege, cross-account access
3. **Data**: Encryption at rest and in transit, Secrets Manager
4. **Application**: Static analysis, dependency scanning, runtime monitoring

### Compliance Features

- **Audit Trail**: All data mutations logged to CloudTrail
- **Encryption**: AES-256 encryption for all stored data
- **Access Control**: Resource-based IAM policies
- **Monitoring**: Real-time security event monitoring

## Contributing

### Code Standards

- **Functional Programming**: Pure functions, immutable data structures
- **DDD Principles**: Clear domain boundaries, bounded contexts
- **Test Coverage**: 95%+ mutation testing coverage required
- **Documentation**: All public APIs documented
- **Security**: Static analysis and dependency scanning

### Pull Request Process

1. Create feature branch from `main`
2. Implement changes following DDD principles
3. Add comprehensive tests (unit, property-based, contract)
4. Run full test suite: `mage ci`
5. Update documentation as needed
6. Submit pull request with clear description

### Plugin Development Guidelines

When creating new plugins:

1. **Follow DDD patterns**: Clear bounded contexts, domain-specific types
2. **Implement proper error handling**: Graceful degradation, retry logic
3. **Add comprehensive testing**: Unit, integration, property-based tests
4. **Document API contracts**: Clear interfaces and expected behaviors
5. **Consider performance**: Connection pooling, caching, async patterns

## Troubleshooting

### Common Issues

1. **Build Failures**: Check Go version (1.24.5+) and run `mage clean && mage build`
2. **Test Failures**: Ensure AWS credentials configured for integration tests
3. **Neptune Connection**: Verify VPC configuration and security groups
4. **Permission Errors**: Check IAM roles and policies for Lambda functions

### Support

For issues and questions:
1. Check build system output: `mage build -v`
2. Review CloudWatch logs for Lambda functions
3. Verify Terraform state and resources
4. Run health checks: API Gateway, Neptune cluster status

## Roadmap

### Phase 2 (Q2 2025)
- **Machine Learning**: Anomaly detection for resource patterns
- **Real-time Streaming**: Kinesis integration for sub-second updates
- **Advanced Analytics**: Athena/QuickSight business intelligence
- **Multi-Cloud Support**: Azure and GCP resource discovery

### Phase 3 (Q3-Q4 2025)  
- **AI-Powered Insights**: Natural language query processing
- **Predictive Analytics**: Cost and capacity forecasting
- **Automation Platform**: Resource lifecycle management
- **Edge Computing**: Regional data processing for global deployments

## License

This project is licensed under the MIT License - see the LICENSE file for details.

---

## Technical Architecture

For detailed technical architecture, implementation patterns, and design decisions, see:
- [DESIGN.md](./DESIGN.md) - Comprehensive technical design document
- [BUILD_SYSTEM.md](./BUILD_SYSTEM.md) - Build system documentation
- [terraform/README.md](./terraform/README.md) - Infrastructure deployment guide