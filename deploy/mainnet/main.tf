# VAJRA Mainnet - AWS ECS Fargate Infrastructure
# This Terraform configuration deploys the VAJRA Relay Chain to an external, production-ready AWS environment.

provider "aws" {
  region = "us-east-1"
}

# 1. Network Topology (VPC & Subnets)
resource "aws_vpc" "vajra_mainnet_vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true
  tags = {
    Name = "vajra-mainnet-vpc"
  }
}

resource "aws_subnet" "public_subnet_1" {
  vpc_id                  = aws_vpc.vajra_mainnet_vpc.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "us-east-1a"
}

# 2. Security Groups (Nginx Rate-Limiting Ingress)
resource "aws_security_group" "relay_sg" {
  name        = "vajra-relay-sg"
  description = "Allow inbound RPC and P2P traffic"
  vpc_id      = aws_vpc.vajra_mainnet_vpc.id

  # HTTP RPC API (Rate limited via Nginx internally)
  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTPS RPC API
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # P2P Node Gossip Protocol
  ingress {
    from_port   = 9000
    to_port     = 9000
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# 3. ECS Cluster & Task Definition (Serverless Docker execution)
resource "aws_ecs_cluster" "vajra_cluster" {
  name = "vajra-mainnet-cluster"
}

resource "aws_ecs_task_definition" "relay_node" {
  family                   = "vajra-relay-task"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 2048
  memory                   = 4096
  execution_role_arn       = aws_iam_role.ecs_execution_role.arn

  container_definitions = jsonencode([
    {
      name      = "relay-node"
      image     = "vajranode/relay-chain:latest"
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
          protocol      = "tcp"
        },
        {
          containerPort = 9000
          hostPort      = 9000
          protocol      = "tcp"
        }
      ]
      environment = [
        { name = "TANTRA_BUCKET_URL", value = "https://insightflow.blackholeinfiverse.com/api/v1/bucket" },
        { name = "ETH_RPC_URL", value = "https://mainnet.infura.io/v3/YOUR_INFURA_KEY" },
        { name = "PDV_STRICT_MODE", value = "true" },
        { name = "SIGVERIFY_STRICT_MODE", value = "true" }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "/ecs/vajra-relay"
          "awslogs-region"        = "us-east-1"
          "awslogs-stream-prefix" = "relay"
        }
      }
    }
  ])
}

# 4. IAM Roles for ECS
resource "aws_iam_role" "ecs_execution_role" {
  name = "vajra_ecs_execution_role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_execution_policy" {
  role       = aws_iam_role.ecs_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Outputs for DNS / DevOps mapping
output "vpc_id" {
  value = aws_vpc.vajra_mainnet_vpc.id
}

output "ecs_cluster_name" {
  value = aws_ecs_cluster.vajra_cluster.name
}
