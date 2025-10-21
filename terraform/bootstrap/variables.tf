// variables.tf - Bootstrap backend variables

variable "backend_bucket_name" {
  description = "Name of the S3 bucket to store Terraform state"
  type        = string
}

variable "backend_region" {
  description = "AWS region for the S3 bucket and DynamoDB table"
  type        = string
  default     = "us-east-1"
}

variable "dynamodb_table_name" {
  description = "Name of the DynamoDB table for Terraform state locking"
  type        = string
}

variable "tags" {
  description = "Optional tags to apply to resources"
  type        = map(string)
  default     = {}
}
