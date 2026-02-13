resource "aws_security_group" "pipeline" {
  name        = "${local.name}-sg"
  description = "Security group for BCD pipeline ECS task"
  vpc_id      = var.vpc_id

  # HTTPS egress - S3, ECR, CloudWatch Logs, WebDAV
  egress {
    description = "HTTPS outbound"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # HTTP egress - WebDAV fallback
  egress {
    description = "HTTP outbound"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.default_tags, {
    Name = "${local.name}-sg"
  })
}
