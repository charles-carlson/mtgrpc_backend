output "instance_public_ip" {
  description = "Public IP of the gRPC server"
  value       = module.ec2.instance_public_ip
}

output "dynamodb_table_name" {
  description = "DynamoDB table name"
  value       = module.dynamodb.user_table_name
}
