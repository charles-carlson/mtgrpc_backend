resource "aws_dynamodb_table" "cards" {
  name         = var.table_name
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "name"
  range_key    = "set_number"

  attribute {
    name = "name"
    type = "S"
  }

  attribute {
    name = "set_number"
    type = "S"
  }
  tags = {
    Name        = "cards table"
    Environment = var.environment
  }
}
