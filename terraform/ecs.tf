resource "aws_cloudwatch_log_group" "pipeline" {
  name              = "/ecs/${local.name}"
  retention_in_days = 30
  tags              = local.default_tags
}

resource "aws_ecs_task_definition" "pipeline" {
  family                   = local.name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.task_cpu
  memory                   = var.task_memory
  execution_role_arn       = aws_iam_role.task_execution.arn
  task_role_arn            = aws_iam_role.task.arn

  ephemeral_storage {
    size_in_gib = var.ephemeral_storage_gb
  }

  container_definitions = jsonencode([{
    name      = local.name
    image     = "${var.ecr_repository_url}:${var.image_tag}"
    essential = true

    environment = [
      { name = "S3_BUCKET", value = var.s3_bucket_name },
      { name = "S3_PREFIX", value = var.s3_prefix },
      { name = "STORAGE_CLASS", value = var.storage_class },
      { name = "CREATE_LATEST_LINK", value = tostring(var.create_latest_link) },
      { name = "CLEANUP_OLD_VERSIONS", value = tostring(var.cleanup_old_versions) },
      { name = "KEEP_VERSIONS", value = tostring(var.keep_versions) },
      { name = "BCD_WORKDIR", value = "/tmp/cnpj_rf" },
      { name = "BCD_MODE", value = var.bcd_mode },
      { name = "BCD_SKIP_VACUUM", value = tostring(var.skip_vacuum) },
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.pipeline.name
        "awslogs-region"        = var.aws_region
        "awslogs-stream-prefix" = "pipeline"
      }
    }
  }])

  tags = local.default_tags
}
