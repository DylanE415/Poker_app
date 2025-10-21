// main.tf - Bootstrap resources for Terraform remote state

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

// Configure AWS provider
provider "aws" {
  region = var.backend_region
}

// S3 bucket to hold Terraform state
resource "aws_s3_bucket" "tfstate" {
  bucket = var.backend_bucket_name
  acl    = "private"

  // Enable versioning for state history
  versioning {
    enabled = true
  }

  // Server-side encryption using AES256 (SSE-S3)
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }

  tags = merge({
    Name = "terraform-state-${var.backend_bucket_name}"
  }, var.tags)
}

// Optional block to enforce bucket-level public access block
resource "aws_s3_bucket_public_access_block" "tfstate_block" {
  bucket = aws_s3_bucket.tfstate.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

// DynamoDB table for state locking
resource "aws_dynamodb_table" "terraform_locks" {
  name         = var.dynamodb_table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  tags = merge({
    Name = "terraform-locks-${var.dynamodb_table_name}"
  }, var.tags)
}
