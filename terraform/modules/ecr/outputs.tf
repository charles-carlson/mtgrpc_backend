output "repository_url" {
  description = "Full URL of the ECR repository"
  value       = aws_ecr_repository.mtgrpc-app.repository_url
}

output "repository_arn" {
  value = aws_ecr_repository.mtgrpc-app.arn
}
