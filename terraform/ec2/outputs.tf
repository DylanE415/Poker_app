output "instance_id" {
  description = "ID of the EC2 instance"
  value       = aws_instance.app.id
}

output "public_ip" {
  description = "Public IP address of the instance"
  value       = aws_instance.app.public_ip
}

output "public_dns" {
  description = "Public DNS name of the instance"
  value       = aws_instance.app.public_dns
}

output "security_group_id" {
  description = "Security group created for the instance"
  value       = aws_security_group.web_sg.id
}

output "iam_role_name" {
  description = "Name of the IAM role attached to the instance"
  value       = aws_iam_role.instance_role.name
}

output "instance_profile" {
  description = "Instance profile name attached to the instance"
  value       = aws_iam_instance_profile.instance_profile.name
}
