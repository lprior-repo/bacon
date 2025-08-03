# Neptune Database Configuration for Beacon Project
# This file defines the Neptune serverless cluster and related resources

# Neptune Cluster
resource "aws_neptune_cluster" "main" {
  cluster_identifier           = local.neptune_cluster_name
  engine                       = "neptune"
  engine_version               = "1.3.3.0"
  backup_retention_period      = var.namespace == "prod" ? 7 : 1
  preferred_backup_window      = "07:00-09:00"
  preferred_maintenance_window = "sun:05:00-sun:06:00"

  # Serverless configuration
  serverless_v2_scaling_configuration {
    max_capacity = var.namespace == "prod" ? 16.0 : 4.0
    min_capacity = 0.5
  }

  # Network configuration
  db_subnet_group_name   = local.neptune_config.subnet_group_name
  vpc_security_group_ids = local.neptune_config.security_group_ids

  # Parameter and cluster parameter groups
  db_cluster_parameter_group_name = aws_neptune_cluster_parameter_group.main.name
  neptune_parameter_group_name    = local.neptune_config.parameter_group_name

  # Security configuration
  storage_encrypted                   = true
  kms_key_id                          = aws_kms_key.neptune.arn
  iam_database_authentication_enabled = true

  # Deletion protection and backup
  deletion_protection       = var.namespace == "prod"
  skip_final_snapshot       = var.namespace != "prod"
  final_snapshot_identifier = var.namespace == "prod" ? "${local.neptune_cluster_name}-final-snapshot-${formatdate("YYYY-MM-DD-hhmm", timestamp())}" : null

  # Logging configuration
  enable_cloudwatch_logs_exports = [
    "audit",
    "slowquery"
  ]

  # Apply changes immediately in non-production environments
  apply_immediately = var.namespace != "prod"

  tags = merge(local.common_tags, {
    Name = local.neptune_cluster_name
    Type = "Database"
  })

  depends_on = [
    aws_cloudwatch_log_group.neptune_audit,
    aws_cloudwatch_log_group.neptune_slowquery
  ]
}

# Neptune Cluster Parameter Group
resource "aws_neptune_cluster_parameter_group" "main" {
  family      = "neptune1.3"
  name        = "${local.name_prefix}-neptune-cluster-params"
  description = "Neptune cluster parameter group for ${local.name_prefix}"

  parameter {
    name  = "neptune_enable_audit_log"
    value = "1"
  }

  parameter {
    name  = "neptune_query_timeout"
    value = "120000" # 2 minutes
  }

  parameter {
    name  = "neptune_result_cache"
    value = "1"
  }

  parameter {
    name  = "neptune_query_force_hard_limit"
    value = "1"
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-cluster-params"
    Type = "Database"
  })
}

# KMS Key for Neptune encryption
resource "aws_kms_key" "neptune" {
  description             = "KMS key for Neptune database encryption"
  deletion_window_in_days = var.namespace == "prod" ? 30 : 7
  enable_key_rotation     = true

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "Enable IAM User Permissions"
        Effect = "Allow"
        Principal = {
          AWS = "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
        }
        Action   = "kms:*"
        Resource = "*"
      },
      {
        Sid    = "Enable Neptune Service Permissions"
        Effect = "Allow"
        Principal = {
          Service = "rds.amazonaws.com"
        }
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:CreateGrant"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "kms:ViaService" = "rds.${data.aws_region.current.name}.amazonaws.com"
          }
        }
      }
    ]
  })

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-kms-key"
    Type = "Security"
  })
}

# KMS Key Alias
resource "aws_kms_alias" "neptune" {
  name          = "alias/${local.name_prefix}-neptune"
  target_key_id = aws_kms_key.neptune.key_id
}

# CloudWatch Log Groups for Neptune
resource "aws_cloudwatch_log_group" "neptune_audit" {
  name              = "/aws/neptune/${local.neptune_cluster_name}/audit"
  retention_in_days = var.namespace == "prod" ? 30 : 14

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-audit-logs"
    Type = "Logging"
  })
}

resource "aws_cloudwatch_log_group" "neptune_slowquery" {
  name              = "/aws/neptune/${local.neptune_cluster_name}/slowquery"
  retention_in_days = var.namespace == "prod" ? 30 : 14

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-slowquery-logs"
    Type = "Logging"
  })
}

# CloudWatch Alarms for Neptune monitoring
resource "aws_cloudwatch_metric_alarm" "neptune_cpu_utilization" {
  alarm_name          = "${local.name_prefix}-neptune-cpu-utilization"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/Neptune"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors Neptune CPU utilization"
  alarm_actions       = [] # Add SNS topic ARN for notifications if needed

  dimensions = {
    DBClusterIdentifier = aws_neptune_cluster.main.cluster_identifier
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-cpu-alarm"
    Type = "Monitoring"
  })
}

resource "aws_cloudwatch_metric_alarm" "neptune_database_connections" {
  alarm_name          = "${local.name_prefix}-neptune-database-connections"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "DatabaseConnections"
  namespace           = "AWS/Neptune"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors Neptune database connections"
  alarm_actions       = [] # Add SNS topic ARN for notifications if needed

  dimensions = {
    DBClusterIdentifier = aws_neptune_cluster.main.cluster_identifier
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-connections-alarm"
    Type = "Monitoring"
  })
}

resource "aws_cloudwatch_metric_alarm" "neptune_gremlin_errors" {
  alarm_name          = "${local.name_prefix}-neptune-gremlin-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "GremlinErrors"
  namespace           = "AWS/Neptune"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "This metric monitors Neptune Gremlin query errors"
  alarm_actions       = [] # Add SNS topic ARN for notifications if needed

  dimensions = {
    DBClusterIdentifier = aws_neptune_cluster.main.cluster_identifier
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-gremlin-errors-alarm"
    Type = "Monitoring"
  })
}

# Neptune IAM role for cluster operations (enhanced permissions)
resource "aws_iam_role_policy" "neptune_enhanced_policy" {
  name = "${local.name_prefix}-neptune-enhanced-policy"
  role = module.neptune_role.iam_role_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "NeptuneClusterAccess"
        Effect = "Allow"
        Action = [
          "neptune-db:*",
          "rds:DescribeDBClusters",
          "rds:DescribeDBInstances",
          "rds:ListTagsForResource"
        ]
        Resource = "*"
      },
      {
        Sid    = "CloudWatchLogsAccess"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams"
        ]
        Resource = [
          "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/neptune/*"
        ]
      },
      {
        Sid    = "KMSAccess"
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
          "kms:CreateGrant"
        ]
        Resource = aws_kms_key.neptune.arn
        Condition = {
          StringEquals = {
            "kms:ViaService" = "rds.${data.aws_region.current.name}.amazonaws.com"
          }
        }
      }
    ]
  })
}

# Local values for Neptune configuration
locals {
  neptune_cluster = {
    cluster_identifier = aws_neptune_cluster.main.cluster_identifier
    endpoint           = aws_neptune_cluster.main.endpoint
    reader_endpoint    = aws_neptune_cluster.main.reader_endpoint
    port               = aws_neptune_cluster.main.port
    arn                = aws_neptune_cluster.main.arn
    cluster_members    = aws_neptune_cluster.main.cluster_members
  }

  neptune_monitoring = {
    cpu_alarm_arn            = aws_cloudwatch_metric_alarm.neptune_cpu_utilization.arn
    connections_alarm_arn    = aws_cloudwatch_metric_alarm.neptune_database_connections.arn
    gremlin_errors_alarm_arn = aws_cloudwatch_metric_alarm.neptune_gremlin_errors.arn
    audit_log_group          = aws_cloudwatch_log_group.neptune_audit.name
    slowquery_log_group      = aws_cloudwatch_log_group.neptune_slowquery.name
  }

  neptune_security = {
    kms_key_id  = aws_kms_key.neptune.key_id
    kms_key_arn = aws_kms_key.neptune.arn
    kms_alias   = aws_kms_alias.neptune.name
  }
}