// EC2 instance, IAM role, security group, and supporting resources

terraform {
  required_version = ">= 1.5.0"
}

// Data: AMI lookup when ami_id not provided
data "aws_ami" "selected" {
  count = var.ami_id == null ? 1 : 0

  most_recent = true

  filter {
    # Default filter map expected to include name pattern if provided
    name   = "name"
    values = [lookup(var.ami_filters, "name", "ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*")]
  }

  owners = length(var.ami_owners) > 0 ? var.ami_owners : ["099720109477"]
}

locals {
  ami_to_use  = var.ami_id != null ? var.ami_id : data.aws_ami.selected[0].id
  merged_tags = merge({ Name = "${var.service_name}-instance" }, var.tags)
}

// IAM assume role policy for EC2
data "aws_iam_policy_document" "assume_ec2" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "instance_role" {
  name               = "${var.service_name}-instance-role"
  assume_role_policy = data.aws_iam_policy_document.assume_ec2.json
  tags               = var.tags
}

// Policy allowing s3:GetObject on the specific object
data "aws_iam_policy_document" "s3_read" {
  statement {
    actions = ["s3:GetObject"]
    resources = [
      "arn:aws:s3:::${var.s3_binary_bucket}/${var.s3_binary_key}"
    ]
    effect = "Allow"
  }
}

resource "aws_iam_role_policy" "s3_access" {
  name   = "${var.service_name}-s3-access"
  role   = aws_iam_role.instance_role.id
  policy = data.aws_iam_policy_document.s3_read.json
}

resource "aws_iam_instance_profile" "instance_profile" {
  name = "${var.service_name}-instance-profile"
  role = aws_iam_role.instance_role.name
}

// Security group allowing HTTP and optionally SSH
resource "aws_security_group" "web_sg" {
  name        = "${var.service_name}-sg"
  description = var.allow_ssh ? "Allow HTTP and SSH" : "Allow HTTP"
  vpc_id      = var.vpc_id

  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = [var.http_cidr]
  }

  dynamic "ingress" {
    for_each = var.allow_ssh ? [1] : []
    content {
      description = "SSH"
      from_port   = 22
      to_port     = 22
      protocol    = "tcp"
      cidr_blocks = [var.ssh_cidr]
    }
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(var.tags, { Name = "${var.service_name}-sg" })
}

// EC2 instance
resource "aws_instance" "app" {
  ami                         = local.ami_to_use
  instance_type               = var.instance_type
  subnet_id                   = var.subnet_id
  vpc_security_group_ids      = [aws_security_group.web_sg.id]
  iam_instance_profile        = aws_iam_instance_profile.instance_profile.name
  associate_public_ip_address = true
  key_name                    = var.key_name

  user_data = templatefile("${path.module}/cloud-init.tpl", {
    bucket       = var.s3_binary_bucket
    key          = var.s3_binary_key
    binary_path  = var.binary_install_path
    service_name = var.service_name
    app_port     = var.app_port
  })

  tags = local.merged_tags
}
