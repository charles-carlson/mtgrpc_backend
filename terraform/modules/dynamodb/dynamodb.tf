resource "aws_dynamodb_table" "cards" {
  name         = "cards"
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
}
