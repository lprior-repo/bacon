# Project-specific variables for the Beacon infrastructure

variable "namespace" {
  description = "Namespace for resource naming (replaces environment concept)"
  type        = string
  default     = "dev"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "beacon"
}

variable "existing_vpc_id" {
  description = "ID of existing VPC to use"
  type        = string
  default     = null
}

variable "existing_private_subnet_ids" {
  description = "List of existing private subnet IDs"
  type        = list(string)
  default     = []
}