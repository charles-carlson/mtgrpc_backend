resource "aws_lb" "nlb" {
  name               = "mtg-grpc-nlb-dev"
  internal           = false
  load_balancer_type = "network"
  subnets            = data.aws_subnets.default.ids
  tags = {
    Name        = "mtg-grpc-nlb"
    Environment = "dev"
  }
}
