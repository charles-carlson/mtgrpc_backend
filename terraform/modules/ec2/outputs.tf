output "instance_public_ip" {
  description = "Public IP of the gRPC server"
  value       = aws_instance.grpc_server.public_ip
}
