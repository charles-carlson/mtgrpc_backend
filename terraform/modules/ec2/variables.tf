variable "instance_type" {
  type        = string
  description = "EC2 instance type"
  default     = "t4g.micro"
}

variable "ssh_key_name" {
  type        = string
  description = "Name of the EC2 key pair for SSH access"
}

variable "allowed_ssh_cidr" {
  type        = string
  description = "CIDR block allowed to SSH into the instance"
  default     = "0.0.0.0/0"
}

variable "environment" {
  type        = string
  description = "Deployment environment"
}

variable "iam_instance_profile" {
  type        = string
  description = "Name of the IAM instance profile to attach"
}

variable "vpc_id" {
  type        = string
  description = "VPC ID for the security group"
}

variable "subnet_id" {
  type        = string
  description = "Subnet ID to launch the instance in"
}

variable "ecr_image_url" {
  type        = string
  description = "Full ECR image URL (e.g. 123456789.dkr.ecr.us-west-1.amazonaws.com/mtg-grpc:latest)"
}

variable "aws_region" {
  type        = string
  description = "AWS region (used for ECR login in user_data)"
}
