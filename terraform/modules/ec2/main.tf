# Latest Amazon Linux 2023 ARM AMI
data "aws_ami" "al2023_arm" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-arm64"]
  }

  filter {
    name   = "architecture"
    values = ["arm64"]
  }
}

resource "aws_security_group" "grpc_server" {
  name        = "mtg-grpc-sg"
  description = "Allow gRPC and SSH"
  vpc_id      = var.vpc_id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.allowed_ssh_cidr]
  }

  ingress {
    description = "gRPC"
    from_port   = 50051
    to_port     = 50051
    protocol    = "tcp"
    cidr_blocks = [var.allowed_ssh_cidr]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name        = "mtg-grpc-sg"
    Environment = var.environment
  }
}

resource "aws_instance" "grpc_server" {
  ami                    = data.aws_ami.al2023_arm.id
  instance_type          = var.instance_type
  key_name               = var.ssh_key_name
  subnet_id              = var.subnet_id
  vpc_security_group_ids = [aws_security_group.grpc_server.id]
  iam_instance_profile   = var.iam_instance_profile

  user_data = <<-EOF
    #!/bin/bash
    dnf update -y
    dnf install -y docker
    dnf install -y amazon-cloudwatch-agent

      cat > /opt/aws/amazon-cloudwatch-agent/etc/amazon-cloudwatch-agent.json << 'CWCONFIG'
      {
        "logs": {
          "logs_collected": {
            "files": {
              "collect_list": [{
                "file_path": "/var/lib/docker/containers/*/*.log",
                "log_group_name": "/ec2/mtg-grpc",
                "log_stream_name": "{instance_id}",
                "timestamp_format": "%Y-%m-%dT%H:%M:%S"
              }]
            }
          }
        }
      }
      CWCONFIG

    systemctl start amazon-cloudwatch-agent
    systemctl enable amazon-cloudwatch-agent
    systemctl start docker
    systemctl enable docker

    ECR_REGISTRY=$(echo "${var.ecr_image_url}" | cut -d'/' -f1)
    aws ecr get-login-password --region ${var.aws_region} | \
      docker login --username AWS --password-stdin $ECR_REGISTRY

    docker pull ${var.ecr_image_url}
    docker run -d --restart=always \
      -p 50051:50051 \
      --name mtg-grpc-server \
      ${var.ecr_image_url}
  EOF

  tags = {
    Name        = "mtg-grpc-server"
    Environment = var.environment
  }
}
