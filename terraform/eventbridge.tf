# EventBridge configuration with scheduled events for scraper coordination
module "eventbridge" {
  source  = "terraform-aws-modules/eventbridge/aws"
  version = "~> 3.0"

  create_bus = true
  bus_name   = "${local.name_prefix}-scraper-bus"

  # Event bus configuration
  event_source_name = "beacon.scraper"
  
  # Rules configuration for scheduled scraping
  rules = {
    github_scraper_schedule = {
      description         = "Trigger GitHub scraper on schedule"
      schedule_expression = "rate(1 hour)"  # Run every hour
      state              = "ENABLED"
    }
    
    datadog_scraper_schedule = {
      description         = "Trigger Datadog scraper on schedule"
      schedule_expression = "rate(30 minutes)"  # Run every 30 minutes
      state              = "ENABLED"
    }
    
    
    processor_schedule = {
      description         = "Trigger data processor on schedule"
      schedule_expression = "rate(15 minutes)"  # Run every 15 minutes
      state              = "ENABLED"
    }
  }

  # Targets configuration - will target Step Functions
  targets = {
    github_scraper_schedule = [
      {
        name            = "TriggerGithubScraper"
        arn             = aws_sfn_state_machine.scraper_workflow.arn
        role_arn        = aws_iam_role.eventbridge_role.arn
        input_transformer = {
          input_paths = {}
          input_template = jsonencode({
            scraper_type = "github"
            timestamp    = "<aws.events.event.ingestion-time>"
          })
        }
      }
    ]
    
    datadog_scraper_schedule = [
      {
        name            = "TriggerDatadogScraper"
        arn             = aws_sfn_state_machine.scraper_workflow.arn
        role_arn        = aws_iam_role.eventbridge_role.arn
        input_transformer = {
          input_paths = {}
          input_template = jsonencode({
            scraper_type = "datadog"
            timestamp    = "<aws.events.event.ingestion-time>"
          })
        }
      }
    ]
    
    
    processor_schedule = [
      {
        name            = "TriggerProcessor"
        arn             = aws_sfn_state_machine.scraper_workflow.arn
        role_arn        = aws_iam_role.eventbridge_role.arn
        input_transformer = {
          input_paths = {}
          input_template = jsonencode({
            scraper_type = "processor"
            timestamp    = "<aws.events.event.ingestion-time>"
          })
        }
      }
    ]
  }

  tags = merge(local.common_tags, {
    Name        = "${local.name_prefix}-eventbridge"
    Service     = "eventbridge"
    Environment = var.namespace
  })
}

# IAM role for EventBridge to invoke Step Functions
resource "aws_iam_role" "eventbridge_role" {
  name = "${local.name_prefix}-eventbridge-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "events.amazonaws.com"
        }
      }
    ]
  })

  tags = local.common_tags
}

# IAM policy for EventBridge role
resource "aws_iam_role_policy" "eventbridge_policy" {
  name = "${local.name_prefix}-eventbridge-policy"
  role = aws_iam_role.eventbridge_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "states:StartExecution"
        ]
        Resource = aws_sfn_state_machine.scraper_workflow.arn
      }
    ]
  })
}

# Step Function state machine for scraper workflow orchestration
resource "aws_sfn_state_machine" "scraper_workflow" {
  name       = "${local.name_prefix}-scraper-workflow"
  role_arn   = aws_iam_role.step_function_role.arn
  definition = templatefile("${path.module}/templates/step_function_definition.json.tpl", {
    lambda_function_arn = "arn:aws:lambda:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:function:${local.lambda_function_name}"
    region             = data.aws_region.current.name
    account_id         = data.aws_caller_identity.current.account_id
  })

  logging_configuration {
    log_destination        = "${aws_cloudwatch_log_group.step_function_logs.arn}:*"
    include_execution_data = true
    level                  = "ERROR"
  }

  tags = local.common_tags
}

# CloudWatch Log Group for Step Functions
resource "aws_cloudwatch_log_group" "step_function_logs" {
  name              = "/aws/stepfunctions/${local.step_function_name}"
  retention_in_days = 14

  tags = local.common_tags
}

# Additional IAM policy for Step Functions to write logs
resource "aws_iam_role_policy" "step_function_logs_policy" {
  name = "${local.step_function_role_name}-logs-policy"
  role = aws_iam_role.step_function_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogDelivery",
          "logs:GetLogDelivery",
          "logs:UpdateLogDelivery",
          "logs:DeleteLogDelivery",
          "logs:ListLogDeliveries",
          "logs:PutResourcePolicy",
          "logs:DescribeResourcePolicies",
          "logs:DescribeLogGroups"
        ]
        Resource = "*"
      }
    ]
  })
}

# CloudWatch alarms for monitoring EventBridge rules
resource "aws_cloudwatch_metric_alarm" "eventbridge_failed_invocations" {
  count = var.namespace == "prod" ? 1 : 0

  alarm_name          = "${local.name_prefix}-eventbridge-failed-invocations"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "FailedInvocations"
  namespace           = "AWS/Events"
  period              = "300"
  statistic           = "Sum"
  threshold           = "1"
  alarm_description   = "This metric monitors EventBridge failed invocations"
  alarm_actions       = []  # Add SNS topic ARN for notifications

  dimensions = {
    RuleName = "${local.name_prefix}-scraper-schedule"
  }

  tags = local.common_tags
}

# Custom EventBridge rule for manual scraper triggers
resource "aws_cloudwatch_event_rule" "manual_scraper_trigger" {
  name        = "${local.name_prefix}-manual-scraper-trigger"
  description = "Manual trigger for scraper workflows"

  event_pattern = jsonencode({
    source      = ["beacon.manual"]
    detail-type = ["Manual Scraper Trigger"]
    detail = {
      scraper_type = ["github", "datadog", "aws", "processor"]
    }
  })

  tags = local.common_tags
}

# Target for manual trigger rule
resource "aws_cloudwatch_event_target" "manual_scraper_target" {
  rule      = aws_cloudwatch_event_rule.manual_scraper_trigger.name
  target_id = "ManualScraperTarget"
  arn       = aws_sfn_state_machine.scraper_workflow.arn
  role_arn  = aws_iam_role.eventbridge_role.arn

  input_transformer {
    input_paths = {
      scraper_type = "$.detail.scraper_type"
    }
    input_template = jsonencode({
      scraper_type = "<scraper_type>"
      timestamp    = "<aws.events.event.ingestion-time>"
      trigger_type = "manual"
    })
  }
}