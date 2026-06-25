output "user_table_name" {
  value = aws_dynamodb_table.cards.name
}

output "user_table_arn" {
  value = aws_dynamodb_table.cards.arn
}
