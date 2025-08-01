# Network Security Groups Configuration
# This file defines security groups for Neptune, AppSync, Lambda, and other components

# Security Group for Neptune Database
resource "aws_security_group" "neptune" {
  name_prefix = "${local.name_prefix}-neptune-"
  description = "Security group for Neptune database cluster"
  vpc_id      = local.vpc_id

  # Allow inbound connections from Lambda functions
  ingress {
    description     = "Neptune access from Lambda"
    from_port       = 8182
    to_port         = 8182
    protocol        = "tcp"
    security_groups = [aws_security_group.lambda.id]
  }

  # Allow inbound connections from AppSync Lambda resolvers
  ingress {
    description     = "Neptune access from AppSync Lambda resolvers"
    from_port       = 8182
    to_port         = 8182
    protocol        = "tcp"
    security_groups = [aws_security_group.appsync_lambda.id]
  }

  # Allow outbound connections (minimal required)
  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-sg"
    Type = "Database"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# Security Group for Lambda Functions (Data Scrapers and Processor)
resource "aws_security_group" "lambda" {
  name_prefix = "${local.name_prefix}-lambda-"
  description = "Security group for Lambda scraper and processor functions"
  vpc_id      = local.vpc_id

  # Allow outbound HTTPS for API calls
  egress {
    description = "HTTPS outbound for API calls"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow outbound HTTP for API calls
  egress {
    description = "HTTP outbound for API calls"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow outbound connection to Neptune
  egress {
    description              = "Neptune database access"
    from_port                = 8182
    to_port                  = 8182
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.neptune.id
  }

  # Allow outbound DNS resolution
  egress {
    description = "DNS resolution"
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-lambda-sg"
    Type = "Compute"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# Security Group for AppSync Lambda Resolvers
resource "aws_security_group" "appsync_lambda" {
  name_prefix = "${local.name_prefix}-appsync-lambda-"
  description = "Security group for AppSync Lambda resolvers"
  vpc_id      = local.vpc_id

  # Allow outbound HTTPS for AWS service calls
  egress {
    description = "HTTPS outbound for AWS services"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow outbound connection to Neptune
  egress {
    description              = "Neptune database access"
    from_port                = 8182
    to_port                  = 8182
    protocol                 = "tcp"
    source_security_group_id = aws_security_group.neptune.id
  }

  # Allow outbound DNS resolution
  egress {
    description = "DNS resolution"
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-appsync-lambda-sg"
    Type = "GraphQL"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# Security Group for VPC Endpoints (if needed)
resource "aws_security_group" "vpc_endpoints" {
  name_prefix = "${local.name_prefix}-vpc-endpoints-"
  description = "Security group for VPC endpoints"
  vpc_id      = local.vpc_id

  # Allow inbound HTTPS from Lambda functions
  ingress {
    description = "HTTPS from Lambda functions"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    security_groups = [
      aws_security_group.lambda.id,
      aws_security_group.appsync_lambda.id
    ]
  }

  # Allow inbound HTTPS from VPC CIDR
  ingress {
    description = "HTTPS from VPC"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [data.aws_vpc.existing.cidr_block]
  }

  # Minimal outbound access
  egress {
    description = "All outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-vpc-endpoints-sg"
    Type = "VPCEndpoint"
  })

  lifecycle {
    create_before_destroy = true
  }
}

# Neptune Subnet Group
resource "aws_neptune_subnet_group" "main" {
  name        = "${local.name_prefix}-neptune-subnet-group"
  description = "Neptune subnet group for ${local.name_prefix}"
  subnet_ids  = local.private_subnet_ids

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-subnet-group"
    Type = "Database"
  })
}

# Neptune Parameter Group
resource "aws_neptune_parameter_group" "main" {
  family      = "neptune1.3"
  name        = "${local.name_prefix}-neptune-params"
  description = "Neptune parameter group for ${local.name_prefix}"

  parameter {
    name  = "neptune_enable_audit_log"
    value = "1"
  }

  parameter {
    name  = "neptune_query_timeout"
    value = "20000"
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-neptune-params"
    Type = "Database"
  })
}

# Local values for networking
locals {
  # Security group mappings
  security_groups = {
    neptune        = aws_security_group.neptune.id
    lambda         = aws_security_group.lambda.id
    appsync_lambda = aws_security_group.appsync_lambda.id
    vpc_endpoints  = aws_security_group.vpc_endpoints.id
  }

  # Neptune configuration
  neptune_config = {
    subnet_group_name    = aws_neptune_subnet_group.main.name
    parameter_group_name = aws_neptune_parameter_group.main.name
    security_group_ids   = [aws_security_group.neptune.id]
  }
}