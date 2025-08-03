# SQS queue configuration for scraper task coordination and dead letter handling
module "scraper_queue" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = "${local.name_prefix}-scraper-queue"

  # Queue configuration
  visibility_timeout_seconds = 300     # 5 minutes - should be >= Lambda timeout
  message_retention_seconds  = 1209600 # 14 days
  max_message_size           = 262144  # 256 KB
  delay_seconds              = 0
  receive_wait_time_seconds  = 20 # Long polling

  # Dead letter queue configuration
  redrive_policy = {
    deadLetterTargetArn = module.scraper_dlq.queue_arn
    maxReceiveCount     = 3
  }

  # Server-side encryption
  kms_master_key_id                 = "alias/aws/sqs"
  kms_data_key_reuse_period_seconds = 300

  # Content-based deduplication for FIFO queues (disabled for standard queue)
  fifo_queue                  = false
  content_based_deduplication = false

  tags = merge(local.common_tags, {
    Name        = "${local.name_prefix}-scraper-queue"
    QueueType   = "scraper-tasks"
    Environment = var.namespace
  })
}

# Dead letter queue for failed messages
module "scraper_dlq" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = "${local.name_prefix}-scraper-dlq"

  # DLQ configuration - longer retention for analysis
  message_retention_seconds = 1209600 # 14 days
  max_message_size          = 262144  # 256 KB

  # Server-side encryption
  kms_master_key_id                 = "alias/aws/sqs"
  kms_data_key_reuse_period_seconds = 300

  tags = merge(local.common_tags, {
    Name        = "${local.name_prefix}-scraper-dlq"
    QueueType   = "dead-letter"
    Environment = var.namespace
  })
}

# Processing results queue for downstream consumers
module "processing_queue" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = "${local.name_prefix}-processing-queue"

  # Queue configuration optimized for processing results
  visibility_timeout_seconds = 180    # 3 minutes
  message_retention_seconds  = 604800 # 7 days
  max_message_size           = 262144 # 256 KB
  delay_seconds              = 0
  receive_wait_time_seconds  = 20 # Long polling

  # Dead letter queue configuration
  redrive_policy = {
    deadLetterTargetArn = module.processing_dlq.queue_arn
    maxReceiveCount     = 5
  }

  # Server-side encryption
  kms_master_key_id                 = "alias/aws/sqs"
  kms_data_key_reuse_period_seconds = 300

  tags = merge(local.common_tags, {
    Name        = "${local.name_prefix}-processing-queue"
    QueueType   = "processing-results"
    Environment = var.namespace
  })
}

# Dead letter queue for processing failures
module "processing_dlq" {
  source  = "terraform-aws-modules/sqs/aws"
  version = "~> 4.0"

  name = "${local.name_prefix}-processing-dlq"

  # DLQ configuration
  message_retention_seconds = 1209600 # 14 days
  max_message_size          = 262144  # 256 KB

  # Server-side encryption
  kms_master_key_id                 = "alias/aws/sqs"
  kms_data_key_reuse_period_seconds = 300

  tags = merge(local.common_tags, {
    Name        = "${local.name_prefix}-processing-dlq"
    QueueType   = "dead-letter"
    Environment = var.namespace
  })
}

# IAM policy for Lambda functions to access SQS queues
resource "aws_iam_role_policy" "lambda_sqs_policy" {
  name = "${local.lambda_role_name}-sqs-policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:GetQueueUrl"
        ]
        Resource = [
          module.scraper_queue.queue_arn,
          module.scraper_dlq.queue_arn,
          module.processing_queue.queue_arn,
          module.processing_dlq.queue_arn
        ]
      }
    ]
  })
}

# CloudWatch alarms for queue monitoring
resource "aws_cloudwatch_metric_alarm" "scraper_queue_dlq_alarm" {
  count = var.namespace == "prod" ? 1 : 0

  alarm_name          = "${local.name_prefix}-scraper-dlq-messages"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ApproximateNumberOfVisibleMessages"
  namespace           = "AWS/SQS"
  period              = "300"
  statistic           = "Average"
  threshold           = "1"
  alarm_description   = "This metric monitors scraper DLQ message count"
  alarm_actions       = [] # Add SNS topic ARN for notifications

  dimensions = {
    QueueName = module.scraper_dlq.queue_name
  }

  tags = local.common_tags
}

resource "aws_cloudwatch_metric_alarm" "processing_queue_dlq_alarm" {
  count = var.namespace == "prod" ? 1 : 0

  alarm_name          = "${local.name_prefix}-processing-dlq-messages"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ApproximateNumberOfVisibleMessages"
  namespace           = "AWS/SQS"
  period              = "300"
  statistic           = "Average"
  threshold           = "1"
  alarm_description   = "This metric monitors processing DLQ message count"
  alarm_actions       = [] # Add SNS topic ARN for notifications

  dimensions = {
    QueueName = module.processing_dlq.queue_name
  }

  tags = local.common_tags
}