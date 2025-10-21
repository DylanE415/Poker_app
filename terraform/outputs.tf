/*
Root outputs aggregating child module outputs for easy consumption.
*/

output "ec2_public_ip" {
  description = "Public IP address of the app EC2 instance."
  value       = module.app_server.public_ip
}

output "ec2_public_dns" {
  description = "Public DNS name of the app EC2 instance."
  value       = module.app_server.public_dns
}

output "s3_website_url" {
  description = "S3 website endpoint for the static assets bucket (HTTP)."
  value       = module.static_assets.website_endpoint
}

output "iam_role_name" {
  description = "IAM role name attached to the EC2 instance."
  value       = module.app_server.iam_role_name
}

output "instance_profile" {
  description = "Instance profile name for the EC2 instance."
  value       = module.app_server.instance_profile
}
