variable "dynamodb_table_arn" {
  type        = string
  description = "ARN of the DynamoDB cards table"
}

variable "ecr_repository_arn" {
  type        = string
  description = "ARN of the ECR repository"
}
