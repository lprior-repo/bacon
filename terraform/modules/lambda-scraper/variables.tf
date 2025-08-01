variable "function_name" {
  description = "Name of the Lambda function"
  type        = string
}

variable "source_path" {
  description = "Path to the Lambda function source code"
  type        = string
}

variable "runtime" {
  description = "Lambda runtime"
  type        = string
  default     = "go1.x"
}

variable "memory_size" {
  description = "Memory size for Lambda function"
  type        = number
  default     = 128
}

variable "timeout" {
  description = "Timeout for Lambda function"
  type        = number
  default     = 30
}

variable "subnet_ids" {
  description = "VPC subnet IDs for Lambda function"
  type        = list(string)
}

variable "security_group_ids" {
  description = "Security group IDs for Lambda function"
  type        = list(string)
}

variable "environment_variables" {
  description = "Environment variables for Lambda function"
  type        = map(string)
  default     = {}
}

variable "dynamodb_table_name" {
  description = "DynamoDB table name"
  type        = string
}

variable "dynamodb_table_arn" {
  description = "DynamoDB table ARN"
  type        = string
}

variable "dynamodb_permissions" {
  description = "Whether to grant DynamoDB permissions"
  type        = bool
  default     = true
}

variable "secrets_arns" {
  description = "ARNs of secrets this function can access"
  type        = list(string)
  default     = []
}

variable "secrets_permissions" {
  description = "Whether to grant Secrets Manager permissions"
  type        = bool
  default     = false
}

variable "additional_sqs_arns" {
  description = "Additional SQS queue ARNs for permissions"
  type        = list(string)
  default     = []
}

variable "sqs_permissions" {
  description = "Whether to grant SQS permissions"
  type        = bool
  default     = true
}

variable "additional_permissions" {
  description = "Additional IAM policy statements"
  type        = list(any)
  default     = []
}

variable "tracing_enabled" {
  description = "Whether to enable X-Ray tracing"
  type        = bool
  default     = true
}

variable "log_retention_days" {
  description = "CloudWatch log retention days"
  type        = number
  default     = 14
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}