terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

data "aws_ecr_authorization_token" "registry" {}
provider "docker" {
  registry_auth {
    address  = data.aws_ecr_authorization_token.registry.proxy_endpoint
    username = data.aws_ecr_authorization_token.registry.user_name
    password = data.aws_ecr_authorization_token.registry.password
  }
}
data "aws_caller_identity" "current" {}

# Default VPC and subnets
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

module "dynamodb" {
  source      = "./modules/dynamodb"
  table_name  = "cards"
  environment = var.environment
}

module "ecr" {
  source              = "./modules/ecr"
  ecr_repository_name = var.ecr_repository_name
  environment         = var.environment
}

module "iam" {
  source             = "./modules/iam"
  dynamodb_table_arn = module.dynamodb.user_table_arn
  ecr_repository_arn = module.ecr.repository_arn
}

module "nlb" {
  source         = "./modules/nlb"
  app_name       = var.service_name
  environment    = var.environment
  vpc_id         = data.aws_vpc.default.id
  subnet_ids     = data.aws_subnets.default.ids
  container_port = 50051
}

module "logging" {
  source       = "./modules/logging"
  service_name = var.service_name
}

module "ec2" {
  source               = "./modules/ec2"
  instance_type        = var.instance_type
  ssh_key_name         = var.ssh_key_name
  allowed_ssh_cidr     = var.allowed_ssh_cidr
  environment          = var.environment
  iam_instance_profile = module.iam.instance_profile_name
  vpc_id               = data.aws_vpc.default.id
  subnet_id            = tolist(data.aws_subnets.default.ids)[0]
  ecr_image_url        = "${module.ecr.repository_url}:latest"
  aws_region           = var.region
}

resource "docker_image" "mtg-grpc-server" {
  name = "${module.ecr.repository_url}:latest"

  build {
    context    = "${path.module}/.."
    dockerfile = "Dockerfile"
    platform   = "linux/arm64"
  }
}

resource "docker_registry_image" "server" {
  name = docker_image.mtg-grpc-server.name
}
