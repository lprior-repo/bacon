# Lambda Functions Configuration for Beacon Project
# This file defines all Lambda functions using terraform-aws-modules/lambda/aws

# GitHub Scraper Lambda Function - Serverless-first design
module "github_scraper_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.13"

  function_name = "${local.name_prefix}-github-scraper"
  description   = "Scrapes GitHub repository data and metadata"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"] # ARM64 for 20% better price/performance

  # Source path pointing to actual Go lambda location
  source_path = "../src/code-analysis/lambda/github-scraper"

  # Serverless-first build configuration
  create_package         = false
  local_existing_package = "../src/code-analysis/lambda/github-scraper/main.zip"

  # Optimized runtime configuration
  timeout                        = 300
  memory_size                    = 512
  reserved_concurrent_executions = 10 # Prevent runaway costs

  # Enhanced ephemeral storage for Go builds
  ephemeral_storage_size = 1024 # 1GB /tmp space

  # VPC configuration with IPv6 support
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.lambda]
  # vpc_config_ipv6_allowed_for_dual_stack = true

  # Environment variables
  environment_variables = {
    DYNAMODB_TABLE          = module.dynamodb_table.dynamodb_table_id
    S3_BUCKET               = module.s3_bucket.s3_bucket_id
    LOG_LEVEL               = "INFO"
    AWS_LAMBDA_EXEC_WRAPPER = "/opt/otel-instrument" # OTEL tracing
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.lambda_scraper

  # Enhanced permissions
  attach_policy_statements = true
  policy_statements = {
    secrets_manager = {
      effect = "Allow"
      actions = [
        "secretsmanager:GetSecretValue"
      ]
      resources = [
        "arn:aws:secretsmanager:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:secret:${local.name_prefix}/github/*"
      ]
    }
    xray_tracing = {
      effect = "Allow"
      actions = [
        "xray:PutTraceSegments",
        "xray:PutTelemetryRecords"
      ]
      resources = ["*"]
    }
  }

  # CloudWatch Logs with JSON formatting
  cloudwatch_logs_retention_in_days = 14
  cloudwatch_logs_log_group_class   = "STANDARD"

  # Advanced logging configuration (not supported in this module version)
  # logging_config = {
  #   log_format            = "JSON"
  #   application_log_level = "INFO"
  #   system_log_level      = "WARN"
  # }

  # X-Ray tracing
  tracing_mode = "Active"

  # SnapStart for faster cold starts (Java/Python only, but future-proofing)
  snap_start = var.environment == "prod" ? "PublishedVersions" : null

  # Event invoke configuration for resilience
  maximum_event_age_in_seconds = 300
  maximum_retry_attempts       = 2

  # Destination configuration
  destination_on_failure = aws_sqs_queue.github_scraper_dlq.arn
  destination_on_success = aws_sns_topic.lambda_success.arn

  # Publish version for blue/green deployments
  publish = true

  # Create alias for deployment strategies  
  # create_current_version_alias = true
  # current_version_alias_name   = "current"

  tags = merge(local.common_tags, {
    Function = "github-scraper"
    Type     = "data-scraper"
    Runtime  = "go"
    Arch     = "arm64"
  })
}

# GitHub Scraper Dead Letter Queue
resource "aws_sqs_queue" "github_scraper_dlq" {
  name = "${local.name_prefix}-github-scraper-dlq"

  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "github-scraper"
    Type     = "dlq"
  })
}

module "github_scraper_dlq" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.github_scraper_dlq.name

  tags = merge(local.common_tags, {
    Function = "github-scraper"
    Type     = "dlq"
  })
}

# DataDog Scraper Lambda Function - Serverless-first design
module "datadog_scraper_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.13"

  function_name = "${local.name_prefix}-datadog-scraper"
  description   = "Scrapes DataDog metrics and monitoring data"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"] # ARM64 for better price/performance

  # Source path pointing to actual Go lambda location
  source_path = "../src/external-integrations/lambda/datadog-scraper"

  # Serverless-first build configuration
  create_package         = false
  local_existing_package = "../src/external-integrations/lambda/datadog-scraper/main.zip"

  # Optimized runtime configuration
  timeout                        = 300
  memory_size                    = 512
  reserved_concurrent_executions = 10

  # Enhanced ephemeral storage
  ephemeral_storage_size = 1024

  # VPC configuration with IPv6 support
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.lambda]
  # vpc_config_ipv6_allowed_for_dual_stack = true

  # Environment variables
  environment_variables = {
    DYNAMODB_TABLE          = module.dynamodb_table.dynamodb_table_id
    S3_BUCKET               = module.s3_bucket.s3_bucket_id
    LOG_LEVEL               = "INFO"
    AWS_LAMBDA_EXEC_WRAPPER = "/opt/otel-instrument"
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.lambda_scraper

  # Enhanced permissions
  attach_policy_statements = true
  policy_statements = {
    secrets_manager = {
      effect = "Allow"
      actions = [
        "secretsmanager:GetSecretValue"
      ]
      resources = [
        "arn:aws:secretsmanager:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:secret:${local.name_prefix}/datadog/*"
      ]
    }
    xray_tracing = {
      effect = "Allow"
      actions = [
        "xray:PutTraceSegments",
        "xray:PutTelemetryRecords"
      ]
      resources = ["*"]
    }
  }

  # CloudWatch Logs with JSON formatting
  cloudwatch_logs_retention_in_days = 14
  cloudwatch_logs_log_group_class   = "STANDARD"

  # Advanced logging configuration (not supported in this module version)
  # logging_config = {
  #   log_format            = "JSON"
  #   application_log_level = "INFO"
  #   system_log_level      = "WARN"
  # }

  # X-Ray tracing
  tracing_mode = "Active"

  # Event invoke configuration for resilience
  maximum_event_age_in_seconds = 300
  maximum_retry_attempts       = 2

  # Destination configuration
  destination_on_failure = aws_sqs_queue.datadog_scraper_dlq.arn
  destination_on_success = aws_sns_topic.lambda_success.arn

  # Publish version for deployment strategies
  publish = true
  # create_current_version_alias = true
  # current_version_alias_name   = "current"

  tags = merge(local.common_tags, {
    Function = "datadog-scraper"
    Type     = "data-scraper"
    Runtime  = "go"
    Arch     = "arm64"
  })
}

# DataDog Scraper Dead Letter Queue
resource "aws_sqs_queue" "datadog_scraper_dlq" {
  name = "${local.name_prefix}-datadog-scraper-dlq"

  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "datadog-scraper"
    Type     = "dlq"
  })
}

module "datadog_scraper_dlq" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.datadog_scraper_dlq.name

  tags = merge(local.common_tags, {
    Function = "datadog-scraper"
    Type     = "dlq"
  })
}

# Note: AWS Scraper removed - replaced by unified Resource Explorer approach in Step Functions

# Processor Lambda Function
module "processor_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-processor"
  description   = "Processes scraped data and stores it in Neptune graph database"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  source_path = "${path.module}/src/processor"

  # Serverless-first build configuration
  create_package         = false
  local_existing_package = "../src/data-processing/lambda/event-processor/main.zip"

  # Runtime configuration
  timeout     = 900  # 15 minutes for data processing
  memory_size = 1024 # More memory for data processing

  # VPC configuration
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.lambda]

  # Environment variables
  environment_variables = {
    DYNAMODB_TABLE   = module.dynamodb_table.dynamodb_table_id
    S3_BUCKET        = module.s3_bucket.s3_bucket_id
    NEPTUNE_ENDPOINT = aws_neptune_cluster.main.endpoint
    NEPTUNE_PORT     = "8182"
    LOG_LEVEL        = "INFO"
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.lambda_processor

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_mode = "Active"

  # Dead letter queue
  dead_letter_target_arn = module.processor_dlq.queue_arn

  tags = merge(local.common_tags, {
    Function = "processor"
    Type     = "data-processor"
  })
}

# Processor Dead Letter Queue
resource "aws_sqs_queue" "processor_dlq" {
  name = "${local.name_prefix}-processor-dlq"

  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 900     # Match Lambda timeout

  tags = merge(local.common_tags, {
    Function = "processor"
    Type     = "dlq"
  })
}

module "processor_dlq" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.processor_dlq.name

  tags = merge(local.common_tags, {
    Function = "processor"
    Type     = "dlq"
  })
}

# CODEOWNERS Scraper Lambda Function
module "codeowners_scraper_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-codeowners-scraper"
  description   = "Scrapes GitHub CODEOWNERS files for ownership data"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  source_path = "${path.module}/src/codeowners_scraper"

  # Serverless-first build configuration
  create_package         = false
  local_existing_package = "../src/data-processing/lambda/event-processor/main.zip"

  # Runtime configuration
  timeout     = 300
  memory_size = 512

  # VPC configuration
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.lambda]

  # Environment variables
  environment_variables = {
    GITHUB_SECRET_ARN = module.secrets.secret_arns["github"]
    DYNAMODB_TABLE    = module.dynamodb_table.dynamodb_table_id
    S3_BUCKET         = module.s3_bucket.s3_bucket_id
    LOG_LEVEL         = "INFO"
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.lambda_scraper

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_mode = "Active"

  # Dead letter queue
  dead_letter_target_arn = module.codeowners_scraper_dlq.queue_arn

  tags = merge(local.common_tags, {
    Function = "codeowners-scraper"
    Type     = "data-scraper"
  })
}

# CODEOWNERS Scraper Dead Letter Queue
resource "aws_sqs_queue" "codeowners_scraper_dlq" {
  name = "${local.name_prefix}-codeowners-scraper-dlq"

  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "codeowners-scraper"
    Type     = "dlq"
  })
}

module "codeowners_scraper_dlq" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.codeowners_scraper_dlq.name

  tags = merge(local.common_tags, {
    Function = "codeowners-scraper"
    Type     = "dlq"
  })
}

# OpenShift Scraper Lambda Function
module "openshift_scraper_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-openshift-scraper"
  description   = "Scrapes OpenShift/Kubernetes metadata for ownership data"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]

  source_path = "${path.module}/src/openshift_scraper"

  # Serverless-first build configuration
  create_package         = false
  local_existing_package = "../src/data-processing/lambda/event-processor/main.zip"

  # Runtime configuration
  timeout     = 300
  memory_size = 512

  # VPC configuration
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.lambda]

  # Environment variables
  environment_variables = {
    DYNAMODB_TABLE = module.dynamodb_table.dynamodb_table_id
    S3_BUCKET      = module.s3_bucket.s3_bucket_id
    LOG_LEVEL      = "INFO"
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.lambda_scraper

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_mode = "Active"

  # Dead letter queue
  dead_letter_target_arn = module.openshift_scraper_dlq.queue_arn

  tags = merge(local.common_tags, {
    Function = "openshift-scraper"
    Type     = "data-scraper"
  })
}

# OpenShift Scraper Dead Letter Queue
resource "aws_sqs_queue" "openshift_scraper_dlq" {
  name = "${local.name_prefix}-openshift-scraper-dlq"

  message_retention_seconds  = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "openshift-scraper"
    Type     = "dlq"
  })
}

module "openshift_scraper_dlq" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.openshift_scraper_dlq.name

  tags = merge(local.common_tags, {
    Function = "openshift-scraper"
    Type     = "dlq"
  })
}

# Local values for Lambda function ARNs and names
locals {
  lambda_functions = {
    github_scraper = {
      arn           = module.github_scraper_lambda.lambda_function_arn
      function_name = module.github_scraper_lambda.lambda_function_name
      invoke_arn    = module.github_scraper_lambda.lambda_function_invoke_arn
    }
    datadog_scraper = {
      arn           = module.datadog_scraper_lambda.lambda_function_arn
      function_name = module.datadog_scraper_lambda.lambda_function_name
      invoke_arn    = module.datadog_scraper_lambda.lambda_function_invoke_arn
    }
    codeowners_scraper = {
      arn           = module.codeowners_scraper_lambda.lambda_function_arn
      function_name = module.codeowners_scraper_lambda.lambda_function_name
      invoke_arn    = module.codeowners_scraper_lambda.lambda_function_invoke_arn
    }
    openshift_scraper = {
      arn           = module.openshift_scraper_lambda.lambda_function_arn
      function_name = module.openshift_scraper_lambda.lambda_function_name
      invoke_arn    = module.openshift_scraper_lambda.lambda_function_invoke_arn
    }
    processor = {
      arn           = module.processor_lambda.lambda_function_arn
      function_name = module.processor_lambda.lambda_function_name
      invoke_arn    = module.processor_lambda.lambda_function_invoke_arn
    }
  }

  dead_letter_queues = {
    github_scraper_dlq     = module.github_scraper_dlq.queue_arn
    datadog_scraper_dlq    = module.datadog_scraper_dlq.queue_arn
    codeowners_scraper_dlq = module.codeowners_scraper_dlq.queue_arn
    openshift_scraper_dlq  = module.openshift_scraper_dlq.queue_arn
    processor_dlq          = module.processor_dlq.queue_arn
  }
}