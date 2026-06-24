resource "aws_iam_role" "ec2_role" {
  name = "mtg-grpc-ec2-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy" "dynamodb_access" {
  name = "mtg-grpc-dynamodb-access"
  role = aws_iam_role.ec2_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:Query",
        "dynamodb:Scan",
      ]
      Resource = aws_dynamodb_table.cards.arn
    }]
  })
}

resource "aws_iam_instance_profile" "ec2_profile" {
  name = "mtg-grpc-ec2-profile"
  role = aws_iam_role.ec2_role.name
}
