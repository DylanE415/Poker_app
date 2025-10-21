/*
Root-level variables for AWS provider configuration and environment inputs.
*/

variable "aws_region" {
  description = "AWS region to target."
  type        = string
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "Optional AWS CLI profile name to use from shared credentials. If null, the SDK default chain is used."
  type        = string
  default     = null
}

variable "assume_role_arn" {
  description = "Optional ARN of an IAM role to assume for provider operations. If null, no assume_role block is configured."
  type        = string
  default     = null
}

variable "assume_role_session_name" {
  description = "Session name to use when assuming a role."
  type        = string
  default     = "tf-session"
}

variable "skip_credentials_validation" {
  description = "If true, skips credentials validation during provider configuration. Useful in some CI scenarios."
  type        = bool
  default     = false
}

# -----------------------------------------------------------------------------
# Environment & infrastructure variables passed down to child modules
# -----------------------------------------------------------------------------

variable "environment" {
  description = "Deployment environment name (e.g., dev, staging, prod)."
  type        = string
  default     = "dev"
}

variable "vpc_id" {
  description = "VPC ID where resources will be created (used by EC2 module)."
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID for the EC2 instance."
  type        = string
}

variable "key_name" {
  description = "Optional EC2 key pair name for SSH access."
  type        = string
  default     = null
}

# Static assets S3 bucket inputs
variable "static_bucket_name" {
  description = "Name of the S3 bucket for static website hosting (must be globally unique)."
  type        = string
}

variable "index_document" {
  description = "Index document for S3 website hosting."
  type        = string
  default     = "index.html"
}

variable "error_document" {
  description = "Error document for S3 website hosting."
  type        = string
  default     = "error.html"
}

variable "force_destroy" {
  description = "Whether to force destroy the S3 bucket when the bucket resource is destroyed."
  type        = bool
  default     = false
}

# Application binary location in S3 (the EC2 instance will download this on boot)
variable "app_binary_bucket" {
  description = "S3 bucket containing the application binary."
  type        = string
}

variable "app_binary_key" {
  description = "S3 key (path) to the application binary object."
  type        = string
}

variable "binary_install_path" {
  description = "Path on the instance where the binary will be installed."
  type        = string
  default     = "/usr/local/bin/myapp"
}

variable "service_name" {
  description = "Name of the service for tagging and IAM naming."
  type        = string
  default     = "myapp"
}

variable "ami_id" {
  description = "Optional explicit AMI id to use for the instance. If null, the module will look up a recent AMI."
  type        = string
  default     = null
}

variable "instance_type" {
  description = "EC2 instance type."
  type        = string
  default     = "t3.micro"
}

variable "allow_ssh" {
  description = "Whether to open SSH (port 22) to the network."
  type        = bool
  default     = false
}

variable "ssh_cidr" {
  description = "CIDR block allowed to SSH when allow_ssh is true. Replace with your IP (e.g. 1.2.3.4/32)."
  type        = string
  default     = "0.0.0.0/0"
}

variable "http_cidr" {
  description = "CIDR block allowed to access HTTP (port 80)."
  type        = string
  default     = "0.0.0.0/0"
}

variable "app_port" {
  description = "Port that the app binds to on localhost."
  type        = number
  default     = 8080
}

variable "tags" {
  description = "A map of tags to apply to resources."
  type        = map(string)
  default     = {}
}
