resource "aws_lambda_function" "scraper" {
  function_name    = var.function_name
  role            = aws_iam_role.lambda_role.arn
  handler         = "main"
  source_code_hash = data.archive_file.lambda_zip.output_base64sha256
  filename        = data.archive_file.lambda_zip.output_path
  runtime         = var.runtime
  memory_size     = var.memory_size
  timeout         = var.timeout

  vpc_config {
    subnet_ids         = var.subnet_ids
    security_group_ids = var.security_group_ids
  }

  environment {
    variables = merge(var.environment_variables, {
      DYNAMODB_TABLE = var.dynamodb_table_name
    })
  }

  dead_letter_config {
    target_arn = aws_sqs_queue.dlq.arn
  }

  tracing_config {
    mode = var.tracing_enabled ? "Active" : "PassThrough"
  }

  depends_on = [
    aws_iam_role_policy_attachment.lambda_basic,
    aws_iam_role_policy_attachment.lambda_vpc,
    aws_cloudwatch_log_group.lambda_logs,
  ]

  tags = var.tags
}

resource "aws_iam_role" "lambda_role" {
  name = "${var.function_name}-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
  role       = aws_iam_role.lambda_role.name
}

resource "aws_iam_role_policy_attachment" "lambda_vpc" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
  role       = aws_iam_role.lambda_role.name
}

resource "aws_iam_role_policy_attachment" "lambda_xray" {
  count      = var.tracing_enabled ? 1 : 0
  policy_arn = "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess"
  role       = aws_iam_role.lambda_role.name
}

resource "aws_iam_role_policy" "lambda_custom" {
  name = "${var.function_name}-custom-policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat(
      var.dynamodb_permissions ? [
        {
          Effect = "Allow"
          Action = [
            "dynamodb:GetItem",
            "dynamodb:PutItem",
            "dynamodb:UpdateItem",
            "dynamodb:DeleteItem",
            "dynamodb:BatchGetItem",
            "dynamodb:BatchWriteItem",
            "dynamodb:TransactGetItems",
            "dynamodb:TransactWriteItems"
          ]
          Resource = var.dynamodb_table_arn
        }
      ] : [],
      var.secrets_permissions ? [
        {
          Effect = "Allow"
          Action = [
            "secretsmanager:GetSecretValue"
          ]
          Resource = var.secrets_arns
        }
      ] : [],
      var.sqs_permissions ? [
        {
          Effect = "Allow"
          Action = [
            "sqs:SendMessage",
            "sqs:ReceiveMessage",
            "sqs:DeleteMessage"
          ]
          Resource = [
            aws_sqs_queue.dlq.arn,
            var.additional_sqs_arns
          ]
        }
      ] : [],
      var.additional_permissions
    )
  })
}

resource "aws_sqs_queue" "dlq" {
  name                      = "${var.function_name}-dlq"
  message_retention_seconds = 1209600 # 14 days
  
  tags = var.tags
}

resource "aws_cloudwatch_log_group" "lambda_logs" {
  name              = "/aws/lambda/${var.function_name}"
  retention_in_days = var.log_retention_days
  
  tags = var.tags
}

data "archive_file" "lambda_zip" {
  type        = "zip"
  source_dir  = var.source_path
  output_path = "${path.module}/../../packages/${var.function_name}.zip"
  depends_on  = [null_resource.build_lambda]
}

resource "null_resource" "build_lambda" {
  triggers = {
    source_hash = filemd5("${var.source_path}/main.go")
  }

  provisioner "local-exec" {
    command = <<-EOF
      cd ${var.source_path}
      GOOS=linux GOARCH=amd64 go build -o main main.go
    EOF
  }
}