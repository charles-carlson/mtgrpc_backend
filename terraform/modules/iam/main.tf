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
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan",
      ]
      Resource = var.dynamodb_table_arn
    }]
  })
}

resource "aws_iam_role_policy" "ecr_pull" {
  name = "mtg-grpc-ecr-pull"
  role = aws_iam_role.ec2_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = [
        "ecr:GetAuthorizationToken",
        "ecr:BatchGetImage",
        "ecr:GetDownloadUrlForLayer",
      ]
      Resource = "*"
    }]
  })
}

resource "aws_iam_instance_profile" "ec2_profile" {
  name = "mtg-grpc-ec2-profile"
  role = aws_iam_role.ec2_role.name
}

resource "aws_iam_role_policy" "cloudwatch_logs" {
  name = "mtg-grpc-cloudwatch-logs"
  role = aws_iam_role.ec2_role.id
  policy = jsonencode(
    {
      Version = "2012-10-17"
      Statement = [{
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams",
        ]
        Resource = "${var.log_group_arn}:*"
      }]
    }
  )
}
