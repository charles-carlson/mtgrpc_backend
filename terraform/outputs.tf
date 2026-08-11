output "instance_public_ip" {
  description = "Public IP of the gRPC server"
  value       = module.ec2.instance_public_ip
}

output "dynamodb_table_name" {
  description = "DynamoDB table name"
  value       = module.dynamodb.user_table_name
}

output "nlb_arn" {
  description = "Feed this into mtg-web/terraform as backend_nlb_arn"
  value       = module.nlb.arn
}

output "nlb_dns_name" {
  description = "Feed this into mtg-web/terraform as backend_nlb_dns_name"
  value       = module.nlb.dns_name
}
