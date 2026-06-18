output "instance_public_ip" {
  description = "Public IP of the gRPC server"
  value       = aws_instance.grpc_server.public_ip
}

output "dynamodb_table_name" {
  description = "DynamoDB table name"
  value       = aws_dynamodb_table.cards.name
}
