variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "bcd-pipeline"
}

variable "ecs_cluster_name" {
  description = "Name of the existing ECS cluster"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID where the ECS tasks will run"
  type        = string
}

variable "subnet_ids" {
  description = "Subnet IDs for the ECS tasks (must have internet access for WebDAV + S3)"
  type        = list(string)
}

variable "ecr_repository_url" {
  description = "ECR repository URL for the pipeline Docker image"
  type        = string
}

variable "image_tag" {
  description = "Docker image tag to deploy"
  type        = string
  default     = "latest"
}

variable "s3_bucket_name" {
  description = "S3 bucket name for storing the SQLite database"
  type        = string
}

variable "s3_prefix" {
  description = "S3 key prefix for the database files"
  type        = string
  default     = "cnpj"
}

variable "storage_class" {
  description = "S3 storage class for uploaded objects"
  type        = string
  default     = "INTELLIGENT_TIERING"
}

variable "create_latest_link" {
  description = "Create a 'latest' symlink in S3 via CopyObject"
  type        = bool
  default     = true
}

variable "cleanup_old_versions" {
  description = "Delete old versions from S3"
  type        = bool
  default     = false
}

variable "keep_versions" {
  description = "Number of versions to keep when cleanup is enabled"
  type        = number
  default     = 3
}

variable "schedule_expression" {
  description = "EventBridge schedule expression (cron or rate)"
  type        = string
  default     = "cron(0 2 5 * ? *)" # Day 5 of each month at 02:00 UTC
}

variable "schedule_timezone" {
  description = "Timezone for the schedule"
  type        = string
  default     = "UTC"
}

variable "task_cpu" {
  description = "Task CPU units (1024 = 1 vCPU)"
  type        = number
  default     = 4096 # 4 vCPU
}

variable "task_memory" {
  description = "Task memory in MiB"
  type        = number
  default     = 8192 # 8 GB
}

variable "ephemeral_storage_gb" {
  description = "Ephemeral storage in GiB (21-200)"
  type        = number
  default     = 200
}

variable "bcd_mode" {
  description = "Download mode: zip or tar"
  type        = string
  default     = "zip"
}

variable "skip_vacuum" {
  description = "Skip VACUUM operation (reduces memory usage)"
  type        = bool
  default     = false
}

variable "assign_public_ip" {
  description = "Assign public IP to the task (required if not using NAT gateway)"
  type        = bool
  default     = true
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}
