/*
Root Terraform versions and required providers.
Pins the AWS provider to ~> 4.0 and requires Terraform >= 1.5.0 for compatibility.
*/

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}
