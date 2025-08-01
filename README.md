# Beacon Infrastructure

This directory contains the core Terraform configuration files for the Beacon project infrastructure on AWS.

## Overview

The Beacon project infrastructure is designed to be deployed in existing AWS VPCs using private subnets for security and cost optimization. The configuration follows AWS and Terraform best practices with modular, reusable components.

## Architecture

- **Compute**: AWS Lambda functions for serverless processing
- **Orchestration**: AWS Step Functions for workflow management
- **Storage**: Amazon S3 for data storage, DynamoDB for state management
- **Networking**: Existing VPC with private subnets for secure deployment
- **Monitoring**: CloudWatch for logging and monitoring

## Files Structure

```
beacon-infra/
├── providers.tf              # Provider configuration and versions
├── variables.tf              # Input variables with validation
├── locals.tf                 # Local values and computed resources
├── main.tf                   # Data sources and main configuration
├── outputs.tf                # Output values
├── modules-examples.tf       # terraform-aws-modules examples (commented)
├── terraform.tfvars.example  # Example variable values
└── README.md                 # This file
```

## Prerequisites

1. **Terraform**: Version >= 1.0
2. **AWS CLI**: Configured with appropriate credentials
3. **Existing Infrastructure**:
   - VPC with private subnets
   - At least 2 private subnets in different AZs for high availability

## Configuration

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `env` | Environment name | `"dev"` |
| `aws_region` | AWS region | `"us-east-1"` |
| `project_name` | Project name | `"beacon"` |
| `existing_vpc_id` | Existing VPC ID | `"vpc-0123456789abcdef0"` |
| `existing_private_subnet_ids` | Private subnet IDs | `["subnet-0123abc", "subnet-4567def"]` |

### Setup

1. **Copy the example variables file**:
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. **Edit terraform.tfvars** with your actual values:
   ```hcl
   env          = "dev"
   aws_region   = "us-east-1"
   project_name = "beacon"
   
   existing_vpc_id = "vpc-your-actual-vpc-id"
   existing_private_subnet_ids = [
     "subnet-your-subnet-1",
     "subnet-your-subnet-2"
   ]
   ```

3. **Initialize Terraform**:
   ```bash
   terraform init
   ```

4. **Validate the configuration**:
   ```bash
   terraform validate
   ```

5. **Plan the deployment**:
   ```bash
   terraform plan
   ```

6. **Apply the configuration**:
   ```bash
   terraform apply
   ```

## Usage with terraform-aws-modules

The `modules-examples.tf` file contains commented examples of how to use popular terraform-aws-modules for:

- **S3 Bucket**: `terraform-aws-modules/s3-bucket/aws`
- **Lambda Function**: `terraform-aws-modules/lambda/aws`
- **Security Groups**: `terraform-aws-modules/security-group/aws`
- **DynamoDB Table**: `terraform-aws-modules/dynamodb-table/aws`
- **IAM Roles**: `terraform-aws-modules/iam/aws`

To use these modules, uncomment the relevant sections in `modules-examples.tf` and customize as needed.

## Naming Convention

All resources follow a consistent naming pattern:
- **Format**: `{project_name}-{env}-{resource_type}`
- **Example**: `beacon-dev-lambda`

## Tagging Strategy

All resources are tagged with:
- `Project`: Project name
- `Environment`: Environment (dev/staging/prod)
- `ManagedBy`: "terraform"
- `CreatedAt`: Timestamp of creation

## Outputs

The configuration provides the following outputs:

- VPC and networking information
- Common tags and naming patterns
- Derived resource names
- AWS account and region information

## Security Considerations

- All resources are deployed in private subnets
- S3 buckets use server-side encryption
- IAM roles follow least privilege principle
- Security groups restrict access appropriately

## Best Practices Implemented

1. **Modularity**: Separate files for different concerns
2. **Validation**: Input validation for all variables
3. **Consistency**: Common naming and tagging patterns
4. **Security**: Private networking and encryption
5. **Maintainability**: Clear documentation and examples

## Pipeline Variables

This configuration is designed to work with CI/CD pipelines using these variables:

```yaml
env: "dev"
aws_region: "us-east-1"
project_name: "beacon"
existing_vpc_id: "vpc-0123456789abcdef0"
existing_private_subnet_ids: ["subnet-0123abc", "subnet-4567def"]
```

## Troubleshooting

### Common Issues

1. **VPC/Subnet Not Found**:
   - Verify the VPC ID and subnet IDs exist in your AWS account
   - Ensure you have proper AWS credentials configured

2. **Permission Errors**:
   - Check that your AWS credentials have sufficient permissions
   - Review IAM policies for Terraform deployment

3. **Validation Errors**:
   - Run `terraform validate` to check syntax
   - Ensure all required variables are provided

### Support

For issues and questions:
1. Check the Terraform validation output
2. Review AWS CloudFormation events if resources fail to create
3. Verify variable values in `terraform.tfvars`

## Contributing

When modifying this configuration:

1. Follow the established naming conventions
2. Add appropriate validation to new variables
3. Update documentation for new features
4. Test changes with `terraform plan` before applying
5. Keep the configuration DRY and modular