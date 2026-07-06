variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-west-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t4g.micro"
}

variable "ssh_key_name" {
  description = "Name of the EC2 key pair for SSH access"
  type        = string
}

variable "allowed_ssh_cidr" {
  description = "CIDR block allowed to SSH into the instance"
  type        = string
}

variable "environment" {
  type        = string
  description = "Deployment environment"
  default     = "dev"
}
variable "service_name" {
  type        = string
  description = "Name of the service"
  default     = "mtg-grpc"
}
variable "ecr_repository_name" {
  type        = string
  description = "Name of the ECR repository"
  default     = "mtg-grpc"
}
