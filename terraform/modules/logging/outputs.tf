output "log_group_name" {
  description = "The CloudWatch log group name"
  value       = aws_cloudwatch_log_group.this.name
}

output "log_group_arn" {
  description = "Cloudwatch log group arn"
  value       = aws_cloudwatch_log_group.this.arn
}
