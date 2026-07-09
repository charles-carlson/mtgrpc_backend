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

  global_secondary_index {
    name            = "set-index"
    hash_key        = "set"
    range_key       = "set_number"
    projection_type = "ALL"
  }
  tags = {
    Name        = "cards table"
    Environment = var.environment
  }
}
