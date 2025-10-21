// outputs.tf - Expose backend bucket and dynamodb table names

output "backend_bucket" {
  description = "S3 bucket created for Terraform backend state"
  value       = aws_s3_bucket.tfstate.bucket
}

output "backend_dynamodb_table" {
  description = "DynamoDB table created for Terraform state locking"
  value       = aws_dynamodb_table.terraform_locks.name
}
