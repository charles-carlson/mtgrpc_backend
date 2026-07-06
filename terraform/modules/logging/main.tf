resource "aws_cloudwatch_log_group" "this" {
  name              = "/ec2/${var.service_name}"
  retention_in_days = var.retention_in_days
}
