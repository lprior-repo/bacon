# Provider configurations only - versions moved to versions.tf
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = local.common_tags
  }
}

provider "random" {
  # Random provider configuration
}