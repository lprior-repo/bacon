# AppSync GraphQL API Configuration for Beacon Project
# This file defines the AppSync GraphQL API with Neptune integration

# AppSync GraphQL API
resource "aws_appsync_graphql_api" "main" {
  authentication_type = "API_KEY"
  name                = local.appsync_api_name
  schema              = file("${path.module}/graphql/schema.graphql")

  # CloudWatch Logs configuration
  log_config {
    cloudwatch_logs_role_arn = local.iam_roles.appsync
    field_log_level          = "ALL"
  }

  # Additional authentication providers
  additional_authentication_provider {
    authentication_type = "AWS_IAM"
  }

  tags = merge(local.common_tags, {
    Name = local.appsync_api_name
    Type = "GraphQL"
  })
}

# AppSync API Key
resource "aws_appsync_api_key" "main" {
  api_id      = aws_appsync_graphql_api.main.id
  description = "API Key for ${local.name_prefix} GraphQL API"
  expires     = "2025-12-31T23:59:59Z"
}

# Neptune Lambda Resolver Function for Queries
module "appsync_query_resolver_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-appsync-query-resolver"
  description   = "AppSync Lambda resolver for Neptune queries"
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/appsync_resolvers/query"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

  # Runtime configuration
  timeout     = 30
  memory_size = 512

  # VPC configuration
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.appsync_lambda]

  # Environment variables
  environment_variables = {
    NEPTUNE_ENDPOINT = local.neptune_cluster.endpoint
    NEPTUNE_PORT     = "8182"
    LOG_LEVEL        = "INFO"
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.appsync_lambda

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_mode = "Active"

  tags = merge(local.common_tags, {
    Function = "appsync-query-resolver"
    Type     = "graphql-resolver"
  })
}

# Neptune Lambda Resolver Function for Mutations
module "appsync_mutation_resolver_lambda" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 7.0"

  function_name = "${local.name_prefix}-appsync-mutation-resolver"
  description   = "AppSync Lambda resolver for Neptune mutations"
  handler       = "main"
  runtime       = "provided.al2023"
  architectures = ["x86_64"]

  source_path = "${path.module}/src/appsync_resolvers/mutation"

  # Build configuration for Go
  build_in_docker = true
  docker_image    = "public.ecr.aws/sam/build-go1.x:latest"

  # Runtime configuration
  timeout     = 30
  memory_size = 512

  # VPC configuration
  vpc_subnet_ids         = local.private_subnet_ids
  vpc_security_group_ids = [local.security_groups.appsync_lambda]

  # Environment variables
  environment_variables = {
    NEPTUNE_ENDPOINT = local.neptune_cluster.endpoint
    NEPTUNE_PORT     = "8182"
    LOG_LEVEL        = "INFO"
  }

  # IAM role configuration
  create_role = false
  lambda_role = local.iam_roles.appsync_lambda

  # CloudWatch Logs configuration
  cloudwatch_logs_retention_in_days = 14

  # X-Ray tracing
  tracing_mode = "Active"

  tags = merge(local.common_tags, {
    Function = "appsync-mutation-resolver"
    Type     = "graphql-resolver"
  })
}

# Lambda Data Source for Query Operations
resource "aws_appsync_datasource" "neptune_query" {
  api_id           = aws_appsync_graphql_api.main.id
  name             = "neptune_query_datasource"
  service_role_arn = local.iam_roles.appsync
  type             = "AWS_LAMBDA"

  lambda_config {
    function_arn = module.appsync_query_resolver_lambda.lambda_function_arn
  }
}

# Lambda Data Source for Mutation Operations
resource "aws_appsync_datasource" "neptune_mutation" {
  api_id           = aws_appsync_graphql_api.main.id
  name             = "neptune_mutation_datasource"
  service_role_arn = local.iam_roles.appsync
  type             = "AWS_LAMBDA"

  lambda_config {
    function_arn = module.appsync_mutation_resolver_lambda.lambda_function_arn
  }
}

# Resolver for getProjects query
resource "aws_appsync_resolver" "get_projects" {
  api_id      = aws_appsync_graphql_api.main.id
  field       = "getProjects"
  type        = "Query"
  data_source = aws_appsync_datasource.neptune_query.name

  request_template = jsonencode({
    version   = "2017-02-28"
    operation = "Invoke"
    payload = {
      field     = "getProjects"
      arguments = "$util.toMap($context.arguments)"
      identity  = "$util.toMap($context.identity)"
    }
  })

  response_template = "$util.toJson($context.result)"
}

# Resolver for getProject query
resource "aws_appsync_resolver" "get_project" {
  api_id      = aws_appsync_graphql_api.main.id
  field       = "getProject"
  type        = "Query"
  data_source = aws_appsync_datasource.neptune_query.name

  request_template = jsonencode({
    version   = "2017-02-28"
    operation = "Invoke"
    payload = {
      field     = "getProject"
      arguments = "$util.toMap($context.arguments)"
      identity  = "$util.toMap($context.identity)"
    }
  })

  response_template = "$util.toJson($context.result)"
}

# Resolver for searchProjects query
resource "aws_appsync_resolver" "search_projects" {
  api_id      = aws_appsync_graphql_api.main.id
  field       = "searchProjects"
  type        = "Query"
  data_source = aws_appsync_datasource.neptune_query.name

  request_template = jsonencode({
    version   = "2017-02-28"
    operation = "Invoke"
    payload = {
      field     = "searchProjects"
      arguments = "$util.toMap($context.arguments)"
      identity  = "$util.toMap($context.identity)"
    }
  })

  response_template = "$util.toJson($context.result)"
}

# Resolver for getMetrics query
resource "aws_appsync_resolver" "get_metrics" {
  api_id      = aws_appsync_graphql_api.main.id
  field       = "getMetrics"
  type        = "Query"
  data_source = aws_appsync_datasource.neptune_query.name

  request_template = jsonencode({
    version   = "2017-02-28"
    operation = "Invoke"
    payload = {
      field     = "getMetrics"
      arguments = "$util.toMap($context.arguments)"
      identity  = "$util.toMap($context.identity)"
    }
  })

  response_template = "$util.toJson($context.result)"
}

# Resolver for createProject mutation
resource "aws_appsync_resolver" "create_project" {
  api_id      = aws_appsync_graphql_api.main.id
  field       = "createProject"
  type        = "Mutation"
  data_source = aws_appsync_datasource.neptune_mutation.name

  request_template = jsonencode({
    version   = "2017-02-28"
    operation = "Invoke"
    payload = {
      field     = "createProject"
      arguments = "$util.toMap($context.arguments)"
      identity  = "$util.toMap($context.identity)"
    }
  })

  response_template = "$util.toJson($context.result)"
}

# Resolver for updateProject mutation
resource "aws_appsync_resolver" "update_project" {
  api_id      = aws_appsync_graphql_api.main.id
  field       = "updateProject"
  type        = "Mutation"
  data_source = aws_appsync_datasource.neptune_mutation.name

  request_template = jsonencode({
    version   = "2017-02-28"
    operation = "Invoke"
    payload = {
      field     = "updateProject"
      arguments = "$util.toMap($context.arguments)"
      identity  = "$util.toMap($context.identity)"
    }
  })

  response_template = "$util.toJson($context.result)"
}

# Resolver for deleteProject mutation
resource "aws_appsync_resolver" "delete_project" {
  api_id      = aws_appsync_graphql_api.main.id
  field       = "deleteProject"
  type        = "Mutation"
  data_source = aws_appsync_datasource.neptune_mutation.name

  request_template = jsonencode({
    version   = "2017-02-28"
    operation = "Invoke"
    payload = {
      field     = "deleteProject"
      arguments = "$util.toMap($context.arguments)"
      identity  = "$util.toMap($context.identity)"
    }
  })

  response_template = "$util.toJson($context.result)"
}

# CloudWatch Log Group for AppSync
resource "aws_cloudwatch_log_group" "appsync" {
  name              = "/aws/appsync/apis/${aws_appsync_graphql_api.main.id}"
  retention_in_days = var.namespace == "prod" ? 30 : 14

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-appsync-logs"
    Type = "Logging"
  })
}

# Local values for AppSync configuration
locals {
  appsync_config = {
    api_id      = aws_appsync_graphql_api.main.id
    api_arn     = aws_appsync_graphql_api.main.arn
    api_key     = aws_appsync_api_key.main.key
    graphql_url = aws_appsync_graphql_api.main.graphql_url
    uris = {
      graphql  = aws_appsync_graphql_api.main.uris["GRAPHQL"]
      realtime = aws_appsync_graphql_api.main.uris["REALTIME"]
    }
  }

  appsync_resolvers = {
    query_lambda_arn    = module.appsync_query_resolver_lambda.lambda_function_arn
    mutation_lambda_arn = module.appsync_mutation_resolver_lambda.lambda_function_arn
    log_group_name      = aws_cloudwatch_log_group.appsync.name
  }
}