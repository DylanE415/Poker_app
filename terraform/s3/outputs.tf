output "website_endpoint" {
  description = "The website endpoint for the S3 bucket (HTTP)."
  value       = aws_s3_bucket.site.website_endpoint
}

output "bucket_arn" {
  description = "ARN of the S3 bucket."
  value       = aws_s3_bucket.site.arn
}

output "bucket_regional_domain_name" {
  description = "Regional domain name for the S3 bucket."
  value       = aws_s3_bucket.site.bucket_regional_domain_name
}
