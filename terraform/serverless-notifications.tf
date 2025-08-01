# Serverless notifications and observability infrastructure
# Following Anton Babenko's serverless.tf patterns

# SNS Topic for Lambda success notifications
resource "aws_sns_topic" "lambda_success" {
  name = "${local.name_prefix}-lambda-success"

  tags = merge(local.common_tags, {
    Purpose = "serverless-notifications"
    Type    = "success-notifications"
  })
}

# SNS Topic for Lambda failure notifications  
resource "aws_sns_topic" "lambda_failures" {
  name = "${local.name_prefix}-lambda-failures"

  tags = merge(local.common_tags, {
    Purpose = "serverless-notifications"
    Type    = "failure-notifications"
  })
}

# CloudWatch Alarms for Lambda functions
resource "aws_cloudwatch_metric_alarm" "lambda_errors" {
  for_each = local.lambda_functions

  alarm_name          = "${each.key}-error-rate"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "This metric monitors lambda errors"
  alarm_actions       = [aws_sns_topic.lambda_failures.arn]

  dimensions = {
    FunctionName = each.value.function_name
  }

  tags = merge(local.common_tags, {
    Function = each.key
    Type     = "error-monitoring"
  })
}

# CloudWatch Alarms for Lambda duration
resource "aws_cloudwatch_metric_alarm" "lambda_duration" {
  for_each = local.lambda_functions

  alarm_name          = "${each.key}-high-duration"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "Duration"
  namespace           = "AWS/Lambda"
  period              = "300"
  statistic           = "Average"
  threshold           = "240000" # 4 minutes (80% of 5-minute timeout)
  alarm_description   = "This metric monitors lambda duration"
  alarm_actions       = [aws_sns_topic.lambda_failures.arn]

  dimensions = {
    FunctionName = each.value.function_name
  }

  tags = merge(local.common_tags, {
    Function = each.key
    Type     = "performance-monitoring"
  })
}

# CloudWatch Dashboard for serverless observability
resource "aws_cloudwatch_dashboard" "serverless_overview" {
  dashboard_name = "${local.name_prefix}-serverless-overview"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6

        properties = {
          metrics = [
            for name, func in local.lambda_functions : [
              "AWS/Lambda",
              "Invocations",
              "FunctionName",
              func.function_name
            ]
          ]
          view    = "timeSeries"
          stacked = false
          region  = data.aws_region.current.name
          title   = "Lambda Invocations"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 12
        height = 6

        properties = {
          metrics = [
            for name, func in local.lambda_functions : [
              "AWS/Lambda",
              "Errors",
              "FunctionName",
              func.function_name
            ]
          ]
          view    = "timeSeries"
          stacked = false
          region  = data.aws_region.current.name
          title   = "Lambda Errors"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 12
        width  = 12
        height = 6

        properties = {
          metrics = [
            for name, func in local.lambda_functions : [
              "AWS/Lambda",
              "Duration",
              "FunctionName",
              func.function_name
            ]
          ]
          view    = "timeSeries"
          stacked = false
          region  = data.aws_region.current.name
          title   = "Lambda Duration"
          period  = 300
        }
      }
    ]
  })

  tags = merge(local.common_tags, {
    Purpose = "serverless-observability"
    Type    = "dashboard"
  })
}

# EventBridge rules for serverless event routing
resource "aws_cloudwatch_event_rule" "lambda_state_changes" {
  name        = "${local.name_prefix}-lambda-state-changes"
  description = "Capture Lambda function state changes"

  event_pattern = jsonencode({
    source      = ["aws.lambda"]
    detail-type = ["Lambda Function State Change"]
    detail = {
      state = ["FAILED", "SUCCEEDED"]
    }
  })

  tags = merge(local.common_tags, {
    Purpose = "serverless-events"
    Type    = "state-monitoring"
  })
}

# EventBridge target for Lambda state changes
resource "aws_cloudwatch_event_target" "lambda_state_sns" {
  rule      = aws_cloudwatch_event_rule.lambda_state_changes.name
  target_id = "SendToSNS"
  arn       = aws_sns_topic.lambda_failures.arn
}