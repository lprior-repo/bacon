# Step Functions Configuration for Beacon Project
# This file defines the Step Function orchestrator using terraform-aws-modules/step-functions/aws

# Step Function State Machine using the terraform-aws-modules/step-functions/aws module
module "beacon_orchestrator_step_function" {
  source  = "terraform-aws-modules/step-functions/aws"
  version = "~> 4.0"

  name       = "${local.name_prefix}-orchestrator"
  definition = templatefile("${path.module}/templates/step_function_definition.json.tpl", {
    github_scraper_arn     = local.lambda_functions.github_scraper.arn
    datadog_scraper_arn    = local.lambda_functions.datadog_scraper.arn
    codeowners_scraper_arn = local.lambda_functions.codeowners_scraper.arn
    openshift_scraper_arn  = local.lambda_functions.openshift_scraper.arn
    processor_arn          = local.lambda_functions.processor.arn
  })

  # Service integration configuration
  service_integrations = {
    lambda = {
      lambda = [
        local.lambda_functions.github_scraper.arn,
        local.lambda_functions.datadog_scraper.arn,
        local.lambda_functions.codeowners_scraper.arn,
        local.lambda_functions.openshift_scraper.arn,
        local.lambda_functions.processor.arn
      ]
    }
  }

  # IAM role configuration
  create_role = false
  role_arn    = local.iam_roles.step_functions

  # CloudWatch Logs configuration
  logging_configuration = {
    level                  = "ALL"
    include_execution_data = true
    log_destination        = "${aws_cloudwatch_log_group.step_functions.arn}:*"
  }

  # X-Ray tracing
  tracing_configuration = {
    enabled = true
  }

  # Type of Step Functions state machine
  type = "STANDARD"

  tags = merge(local.common_tags, {
    Name     = "${local.name_prefix}-orchestrator"
    Type     = "Orchestration"
    Function = "data-processing"
  })

  depends_on = [
    aws_cloudwatch_log_group.step_functions,
    module.github_scraper_lambda,
    module.datadog_scraper_lambda,
    module.processor_lambda
  ]
}

# CloudWatch Log Group for Step Functions
resource "aws_cloudwatch_log_group" "step_functions" {
  name              = "/aws/stepfunctions/${local.name_prefix}-orchestrator"
  retention_in_days = var.env == "prod" ? 30 : 14

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-stepfunctions-logs"
    Type = "Logging"
  })
}

# CloudWatch Alarms for Step Functions monitoring
resource "aws_cloudwatch_metric_alarm" "step_function_execution_failed" {
  alarm_name          = "${local.name_prefix}-stepfunctions-execution-failed"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ExecutionsFailed"
  namespace           = "AWS/States"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "This metric monitors failed Step Function executions"
  alarm_actions       = [] # Add SNS topic ARN for notifications if needed

  dimensions = {
    StateMachineArn = module.beacon_orchestrator_step_function.state_machine_arn
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-stepfunctions-failed-alarm"
    Type = "Monitoring"
  })
}

resource "aws_cloudwatch_metric_alarm" "step_function_execution_timeout" {
  alarm_name          = "${local.name_prefix}-stepfunctions-execution-timeout"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "ExecutionsTimedOut"
  namespace           = "AWS/States"
  period              = "300"
  statistic           = "Sum"
  threshold           = "0"
  alarm_description   = "This metric monitors timed out Step Function executions"
  alarm_actions       = [] # Add SNS topic ARN for notifications if needed

  dimensions = {
    StateMachineArn = module.beacon_orchestrator_step_function.state_machine_arn
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-stepfunctions-timeout-alarm"
    Type = "Monitoring"
  })
}

resource "aws_cloudwatch_metric_alarm" "step_function_execution_throttled" {
  alarm_name          = "${local.name_prefix}-stepfunctions-execution-throttled"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ExecutionsThrottled"
  namespace           = "AWS/States"
  period              = "300"
  statistic           = "Sum"
  threshold           = "5"
  alarm_description   = "This metric monitors throttled Step Function executions"
  alarm_actions       = [] # Add SNS topic ARN for notifications if needed

  dimensions = {
    StateMachineArn = module.beacon_orchestrator_step_function.state_machine_arn
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-stepfunctions-throttled-alarm"
    Type = "Monitoring"
  })
}

# Local values for Step Functions configuration
locals {
  step_function_config = {
    arn                    = module.beacon_orchestrator_step_function.state_machine_arn
    name                   = module.beacon_orchestrator_step_function.state_machine_name
    creation_date          = module.beacon_orchestrator_step_function.state_machine_creation_date
    status                 = module.beacon_orchestrator_step_function.state_machine_status
    log_group_name         = aws_cloudwatch_log_group.step_functions.name
  }

  step_function_monitoring = {
    failed_alarm_arn     = aws_cloudwatch_metric_alarm.step_function_execution_failed.arn
    timeout_alarm_arn    = aws_cloudwatch_metric_alarm.step_function_execution_timeout.arn
    throttled_alarm_arn  = aws_cloudwatch_metric_alarm.step_function_execution_throttled.arn
  }
}