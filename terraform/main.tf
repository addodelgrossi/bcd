terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

locals {
  name = var.project_name

  default_tags = merge(var.tags, {
    Project   = "bcd"
    ManagedBy = "terraform"
  })
}

data "aws_ecs_cluster" "this" {
  cluster_name = var.ecs_cluster_name
}

data "aws_vpc" "this" {
  id = var.vpc_id
}

data "aws_caller_identity" "current" {}
