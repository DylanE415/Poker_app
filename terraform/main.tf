/*
Root orchestration module

This root module wires together the child modules in ./s3 and ./ec2.

Bootstrap / Backend notes (read before initializing):
1. cd ./bootstrap && terraform init && terraform apply
   - This will create an S3 bucket and DynamoDB table used for the Terraform backend.
2. From the bootstrap outputs, run terraform init in the root using backend-config arguments:
   terraform init \
     -backend-config="bucket=<bootstrap-backend-bucket>" \
     -backend-config="key=envs/dev/terraform.tfstate" \
     -backend-config="region=<region>" \
     -backend-config="dynamodb_table=<bootstrap-lock-table>"
   Alternatively, copy the created bootstrap values into a backend block (NOT recommended in VCS).
3. terraform plan && terraform apply

Bootstrap must run before root init to avoid the chicken-and-egg problem creating the backend.
*/

// NOTE: We intentionally do NOT include a backend block here. The operator should run
// `terraform init -backend-config="..."` with the values produced by ./bootstrap outputs.
// If you prefer, uncomment and fill the backend block below with bootstrap outputs (replace <> placeholders):

/*
terraform {
  backend "s3" {
    bucket         = "<bootstrap-backend-bucket>"
    key            = "envs/dev/terraform.tfstate"
    region         = "<region>"
    dynamodb_table = "<bootstrap-lock-table>"
  }
}
*/

// Module: static assets S3 bucket
module "static_assets" {
  source = "./s3"

  bucket_name    = var.static_bucket_name
  index_document = var.index_document
  error_document = var.error_document
  force_destroy  = var.force_destroy
  tags           = var.tags
}

// Module: application EC2 server
module "app_server" {
  source = "./ec2"

  ami_id             = var.ami_id
  instance_type      = var.instance_type
  subnet_id          = var.subnet_id
  vpc_id             = var.vpc_id
  key_name           = var.key_name
  s3_binary_bucket   = var.app_binary_bucket
  s3_binary_key      = var.app_binary_key
  binary_install_path = var.binary_install_path
  service_name       = var.service_name
  allow_ssh          = var.allow_ssh
  ssh_cidr           = var.ssh_cidr
  http_cidr          = var.http_cidr
  app_port           = var.app_port
  tags               = var.tags

  # Ensure the EC2 instance creation depends on the S3 static bucket when desired
  # (e.g., if the app will serve assets from the static bucket). This is optional.
  depends_on = [module.static_assets]
}
