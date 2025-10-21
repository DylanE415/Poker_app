// Variables for EC2 instance module

variable "ami_filters" {
  description = "Map of filters to find the AMI. Example: { name = \"ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*\", owner = \"099720109477\" }"
  type        = map(string)
  default     = {}
}

variable "ami_owners" {
  description = "List of owners to filter AMI by (e.g., [\"099720109477\"] for Canonical). If empty, will use most recent marketplace AMI lookup."
  type        = list(string)
  default     = []
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "subnet_id" {
  description = "Subnet ID where to launch the instance"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for the security group"
  type        = string
}

variable "key_name" {
  description = "Optional key pair name for SSH access"
  type        = string
  default     = null
}

variable "s3_binary_bucket" {
  description = "S3 bucket containing the binary"
  type        = string
}

variable "s3_binary_key" {
  description = "S3 key (path) to the binary object"
  type        = string
}

variable "binary_install_path" {
  description = "Path on the instance where the binary will be installed"
  type        = string
  default     = "/usr/local/bin/myapp"
}

variable "service_name" {
  description = "Name of the systemd service to create for the binary"
  type        = string
  default     = "myapp"
}

variable "allow_ssh" {
  description = "Whether to open SSH (port 22) to the network"
  type        = bool
  default     = false
}

variable "ssh_cidr" {
  description = "CIDR block allowed to SSH when allow_ssh is true"
  type        = string
  default     = "0.0.0.0/0"
}

variable "http_cidr" {
  description = "CIDR block allowed to access HTTP (port 80)"
  type        = string
  default     = "0.0.0.0/0"
}

variable "app_port" {
  description = "Port that the app binds to on localhost"
  type        = number
  default     = 8080
}

variable "ami_id" {
  description = "Optional explicit AMI id. If set, ami_filters and ami_owners are ignored."
  type        = string
  default     = null
}

variable "tags" {
  description = "Tags to apply to resources"
  type        = map(string)
  default     = {}
}
