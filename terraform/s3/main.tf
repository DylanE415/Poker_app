terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

# Configure the AWS provider in the root module when using this as a root; modules should not declare provider blocks.

# S3 bucket for static website hosting
resource "aws_s3_bucket" "site" {
  bucket        = var.bucket_name
  acl           = "private" # keep bucket ACL private, we expose objects via policy
  force_destroy = var.force_destroy

  website {
    index_document = var.index_document
    error_document = var.error_document
  }

  tags = merge({
    Name = "static-assets-${var.bucket_name}"
  }, var.tags)

  # Example lifecycle rule: expire objects under the "tmp/" prefix after 7 days
  lifecycle_rule {
    id      = "tmp-expiration"
    enabled = true

    prefix = "tmp/"

    expiration {
      days = 7
    }
  }
}

# Allow the bucket policy to make objects public by disabling the public access blocks at the bucket level
resource "aws_s3_bucket_public_access_block" "site" {
  bucket = aws_s3_bucket.site.id

  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

# Public read policy for website objects
data "aws_iam_policy_document" "public_read" {
  statement {
    sid = "PublicReadGetObject"

    principals {
      type        = "AWS"
      identifiers = ["*"]
    }

    actions = [
      "s3:GetObject"
    ]

    resources = ["${aws_s3_bucket.site.arn}/*"]
  }
}

resource "aws_s3_bucket_policy" "public" {
  bucket = aws_s3_bucket.site.id
  policy = data.aws_iam_policy_document.public_read.json
}
