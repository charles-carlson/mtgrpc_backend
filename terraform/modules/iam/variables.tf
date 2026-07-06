variable "dynamodb_table_arn" {
  type        = string
  description = "ARN of the DynamoDB cards table"
}

variable "ecr_repository_arn" {
  type        = string
  description = "ARN of the ECR repository"
}

variable "log_group_arn" {
  description = "ARN of the Cloudwatch log group"
  type        = string
}
