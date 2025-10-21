# Root terraform variable assignments (placeholders) - replace values before applying

# AWS region to target
aws_region = "us-east-1"

# Optional AWS CLI profile name; set to null to use default SDK chain
aws_profile = null

# Optional role ARN to assume for provider operations; set to null to skip assume_role
assume_role_arn          = null
assume_role_session_name = "tf-session"

# Environment name
environment = "dev"

# VPC and Subnet where EC2 will be created - REQUIRED: replace with your IDs
vpc_id    = "vpc-REPLACE_ME"
subnet_id = "subnet-REPLACE_ME"

# Optional EC2 key pair name for SSH access. Set to null to skip
key_name = null

# Static assets S3 bucket name - must be globally unique
static_bucket_name = "my-static-bucket-REPLACE_ME"

# Application binary location in S3 (bucket & key) used by EC2 user-data to download the app
app_binary_bucket = "my-app-bucket-REPLACE_ME"
app_binary_key    = "path/to/myapp.tar.gz"

# Where to install the binary on the EC2 instance
binary_install_path = "/usr/local/bin/myapp"

# Service and instance configuration
service_name  = "myapp"
ami_id        = null # optional: set a specific AMI id (e.g., "ami-0123456789abcdef0")
instance_type = "t3.micro"

# Networking / security
allow_ssh = false
ssh_cidr  = "0.0.0.0/0"
http_cidr = "0.0.0.0/0"
app_port  = 8080

# Tags map (example)
tags = {
  Project     = "example"
  Environment = "dev"
}
