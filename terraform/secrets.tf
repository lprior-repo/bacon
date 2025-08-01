# AWS Secrets Manager secrets for GitHub and Datadog API keys
module "github_secrets" {
  source  = "terraform-aws-modules/secrets-manager/aws"
  version = "~> 1.0"

  # Secret for GitHub API token
  name        = "${local.name_prefix}-github-api-token"
  description = "GitHub API token for scraping repository data"

  # Enable rotation for production environments
  rotation_enabled = var.env == "prod"

  # Automatic rotation configuration (30 days for production)
  rotation_rules = var.env == "prod" ? [
    {
      automatically_after_days = 30
    }
  ] : []

  # Secret value can be set via AWS CLI or console after creation
  create_policy = true

  policy_statements = [
    {
      sid = "AllowLambdaAccess"
      principals = [
        {
          type        = "AWS"
          identifiers = [aws_iam_role.lambda_role.arn]
        }
      ]
      actions = [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ]
      resources = ["*"]
    }
  ]

  # Replica configuration for cross-region disaster recovery
  replica = var.env == "prod" ? {
    region = "us-west-2"
  } : {}

  tags = merge(local.common_tags, {
    Name        = "${local.name_prefix}-github-api-token"
    SecretType  = "github-api"
    Rotation    = var.env == "prod" ? "enabled" : "disabled"
  })
}

module "datadog_secrets" {
  source  = "terraform-aws-modules/secrets-manager/aws"
  version = "~> 1.0"

  # Secret for Datadog API key
  name        = "${local.name_prefix}-datadog-api-key"
  description = "Datadog API key for metrics and monitoring integration"

  # Enable rotation for production environments
  rotation_enabled = var.env == "prod"

  # Automatic rotation configuration (90 days for production)
  rotation_rules = var.env == "prod" ? [
    {
      automatically_after_days = 90
    }
  ] : []

  # Secret value can be set via AWS CLI or console after creation
  create_policy = true

  policy_statements = [
    {
      sid = "AllowLambdaAccess"
      principals = [
        {
          type        = "AWS"
          identifiers = [aws_iam_role.lambda_role.arn]
        }
      ]
      actions = [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ]
      resources = ["*"]
    }
  ]

  # Replica configuration for cross-region disaster recovery
  replica = var.env == "prod" ? {
    region = "us-west-2"
  } : {}

  tags = merge(local.common_tags, {
    Name        = "${local.name_prefix}-datadog-api-key"
    SecretType  = "datadog-api"
    Rotation    = var.env == "prod" ? "enabled" : "disabled"
  })
}

# IAM policy attachment for Lambda to access secrets
resource "aws_iam_role_policy" "lambda_secrets_policy" {
  name = "${local.lambda_role_name}-secrets-policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = [
          module.github_secrets.secret_arn,
          module.datadog_secrets.secret_arn
        ]
      }
    ]
  })
}