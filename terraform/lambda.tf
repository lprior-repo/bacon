# Lambda Functions Configuration for Beacon Project
# This file defines all Lambda functions using terraform-aws-modules/lambda/aws

# GitHub Scraper Lambda Function
module "github_scraper_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-github-scraper"
  description   = "Scrapes GitHub repository data and metadata"
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/github_scraper"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

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

  # Permissions for Secrets Manager
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
  }

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_config_mode = "Active"

  # Dead letter queue
  dead_letter_target_arn = module.github_scraper_dlq.queue_arn

  tags = merge(local.common_tags, {
    Function = "github-scraper"
    Type     = "data-scraper"
  })
}

# GitHub Scraper Dead Letter Queue
resource "aws_sqs_queue" "github_scraper_dlq" {
  name = "${local.name_prefix}-github-scraper-dlq"

  message_retention_seconds = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "github-scraper"
    Type     = "dlq"
  })
}

module "github_scraper_dlq" {
  source = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.github_scraper_dlq.name

  tags = merge(local.common_tags, {
    Function = "github-scraper"
    Type     = "dlq"
  })
}

# DataDog Scraper Lambda Function
module "datadog_scraper_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-datadog-scraper"
  description   = "Scrapes DataDog metrics and monitoring data"
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/datadog_scraper"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

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

  # Permissions for Secrets Manager
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
  }

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_config_mode = "Active"

  # Dead letter queue
  dead_letter_target_arn = module.datadog_scraper_dlq.queue_arn

  tags = merge(local.common_tags, {
    Function = "datadog-scraper"
    Type     = "data-scraper"
  })
}

# DataDog Scraper Dead Letter Queue
resource "aws_sqs_queue" "datadog_scraper_dlq" {
  name = "${local.name_prefix}-datadog-scraper-dlq"

  message_retention_seconds = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "datadog-scraper"
    Type     = "dlq"
  })
}

module "datadog_scraper_dlq" {
  source = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.datadog_scraper_dlq.name

  tags = merge(local.common_tags, {
    Function = "datadog-scraper"
    Type     = "dlq"
  })
}

# AWS Scraper Lambda Function
module "aws_scraper_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-aws-scraper"
  description   = "Scrapes AWS service metrics and configuration data"
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/aws_scraper"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

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

  # Permissions for Secrets Manager
  attach_policy_statements = true
  policy_statements = {
    secrets_manager = {
      effect = "Allow"
      actions = [
        "secretsmanager:GetSecretValue"
      ]
      resources = [
        "arn:aws:secretsmanager:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:secret:${local.name_prefix}/aws/*"
      ]
    }
  }

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_config_mode = "Active"

  # Dead letter queue
  dead_letter_target_arn = module.aws_scraper_dlq.queue_arn

  tags = merge(local.common_tags, {
    Function = "aws-scraper"
    Type     = "data-scraper"
  })
}

# AWS Scraper Dead Letter Queue
resource "aws_sqs_queue" "aws_scraper_dlq" {
  name = "${local.name_prefix}-aws-scraper-dlq"

  message_retention_seconds = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "aws-scraper"
    Type     = "dlq"
  })
}

module "aws_scraper_dlq" {
  source = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = aws_sqs_queue.aws_scraper_dlq.name

  tags = merge(local.common_tags, {
    Function = "aws-scraper"
    Type     = "dlq"
  })
}

# Processor Lambda Function
module "processor_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-processor"
  description   = "Processes scraped data and stores it in Neptune graph database"
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/processor"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

  # Runtime configuration
  timeout     = 900  # 15 minutes for data processing
  memory_size = 1024 # More memory for data processing

  # VPC configuration
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.lambda]

  # Environment variables
  environment_variables = {
    DYNAMODB_TABLE     = module.dynamodb_table.dynamodb_table_id
    S3_BUCKET          = module.s3_bucket.s3_bucket_id
    NEPTUNE_ENDPOINT   = aws_neptune_cluster.main.endpoint
    NEPTUNE_PORT       = "8182"
    LOG_LEVEL          = "INFO"
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.lambda_processor

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_config_mode = "Active"

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

  message_retention_seconds = 1209600 # 14 days
  visibility_timeout_seconds = 900    # Match Lambda timeout

  tags = merge(local.common_tags, {
    Function = "processor"
    Type     = "dlq"
  })
}

module "processor_dlq" {
  source = "terraform-aws-modules/sqs/aws"
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
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/codeowners_scraper"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

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
  tracing_config_mode = "Active"

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

  message_retention_seconds = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "codeowners-scraper"
    Type     = "dlq"
  })
}

module "codeowners_scraper_dlq" {
  source = "terraform-aws-modules/sqs/aws"
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
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/openshift_scraper"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

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
  tracing_config_mode = "Active"

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

  message_retention_seconds = 1209600 # 14 days
  visibility_timeout_seconds = 300

  tags = merge(local.common_tags, {
    Function = "openshift-scraper"
    Type     = "dlq"
  })
}

module "openshift_scraper_dlq" {
  source = "terraform-aws-modules/sqs/aws"
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
    aws_scraper = {
      arn           = module.aws_scraper_lambda.lambda_function_arn
      function_name = module.aws_scraper_lambda.lambda_function_name
      invoke_arn    = module.aws_scraper_lambda.lambda_function_invoke_arn
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
    aws_scraper_dlq        = module.aws_scraper_dlq.queue_arn
    codeowners_scraper_dlq = module.codeowners_scraper_dlq.queue_arn
    openshift_scraper_dlq  = module.openshift_scraper_dlq.queue_arn
    processor_dlq          = module.processor_dlq.queue_arn
  }
}