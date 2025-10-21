Bootstrap module to create S3 backend bucket and DynamoDB lock table

Usage:
1. Set values for variables (e.g., via CLI -var or terraform.tfvars):
   - backend_bucket_name
   - dynamodb_table_name
   - backend_region (optional, defaults to us-east-1)
2. Run:
   terraform init
   terraform apply -var='backend_bucket_name=my-terraform-state-bucket' -var='dynamodb_table_name=my-terraform-locks'

After apply, use the outputs to configure the backend in the main root, for example:

terraform {
  backend "s3" {
    bucket         = "<backend_bucket>"
    key            = "envs/prod/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "<backend_dynamodb_table>"
    encrypt        = true
  }
}
