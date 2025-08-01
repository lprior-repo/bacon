# This file contains examples of terraform-aws-modules usage for the Beacon project
# These are commented out templates that can be uncommented and customized as needed

# Example: S3 bucket using terraform-aws-modules/s3-bucket/aws
/*
module "s3_bucket" {
  source = "terraform-aws-modules/s3-bucket/aws"
  version = "~> 4.0"

  bucket = local.s3_bucket_name
  
  # Prevent public access
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true

  # Enable versioning
  versioning = {
    enabled = true
  }

  # Server-side encryption
  server_side_encryption_configuration = {
    rule = {
      apply_server_side_encryption_by_default = {
        sse_algorithm = "AES256"
      }
    }
  }

  tags = local.common_tags
}
*/

# Example: Lambda function using terraform-aws-modules/lambda/aws
/*
module "lambda_function" {
  source = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = local.lambda_function_name
  description   = "Lambda function for ${var.project_name} ${var.env}"
  handler       = "main.lambda_handler"
  runtime       = "python3.11"
  timeout       = 300

  # Source code
  source_path = "../src"

  # VPC configuration
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [aws_security_group.lambda.id]
  attach_network_policy  = true

  # Environment variables
  environment_variables = {
    ENV           = var.env
    PROJECT_NAME  = var.project_name
    S3_BUCKET     = local.s3_bucket_name
    DYNAMODB_TABLE = local.dynamodb_table_name
  }

  # IAM
  attach_policy_statements = true
  policy_statements = {
    dynamodb = {
      effect = "Allow"
      actions = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan"
      ]
      resources = [aws_dynamodb_table.main.arn]
    }
    s3 = {
      effect = "Allow"
      actions = [
        "s3:GetObject",
        "s3:PutObject",
        "s3:DeleteObject"
      ]
      resources = ["${module.s3_bucket.s3_bucket_arn}/*"]
    }
  }

  tags = local.common_tags
}
*/

# Example: Security Group using terraform-aws-modules/security-group/aws
/*
module "lambda_security_group" {
  source = "terraform-aws-modules/security-group/aws"
  version = "~> 5.0"

  name        = "${local.name_prefix}-lambda-sg"
  description = "Security group for Lambda function"
  vpc_id      = local.vpc_id

  # Egress rules
  egress_rules = ["all-all"]

  tags = local.common_tags
}
*/

# Example: DynamoDB table using terraform-aws-modules/dynamodb-table/aws
/*
module "dynamodb_table" {
  source = "terraform-aws-modules/dynamodb-table/aws"
  version = "~> 4.0"

  name           = local.dynamodb_table_name
  hash_key       = "id"
  billing_mode   = "PAY_PER_REQUEST"

  attributes = [
    {
      name = "id"
      type = "S"
    }
  ]

  server_side_encryption_enabled = true

  tags = local.common_tags
}
*/

# Example: IAM role using terraform-aws-modules/iam/aws//modules/iam-role-for-service
/*
module "lambda_role" {
  source = "terraform-aws-modules/iam/aws//modules/iam-role-for-service"
  version = "~> 5.0"

  trusted_role_services = ["lambda.amazonaws.com"]

  role_name         = local.lambda_role_name
  role_description  = "IAM role for ${local.lambda_function_name}"
  role_requires_mfa = false

  custom_role_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole",
  ]

  tags = local.common_tags
}
*/