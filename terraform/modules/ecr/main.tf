resource "aws_ecr_repository" "mtgrpc-app" {
  name                 = var.ecr_repository_name
  image_tag_mutability = "MUTABLE"
  force_delete         = true
  image_scanning_configuration {
    scan_on_push = true
  }
  tags = {
    Name        = var.ecr_repository_name
    Environment = var.environment
  }
}
