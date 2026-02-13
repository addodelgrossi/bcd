output "task_definition_arn" {
  description = "ARN of the ECS task definition"
  value       = aws_ecs_task_definition.pipeline.arn
}

output "task_execution_role_arn" {
  description = "ARN of the task execution role"
  value       = aws_iam_role.task_execution.arn
}

output "task_role_arn" {
  description = "ARN of the task role (S3 access)"
  value       = aws_iam_role.task.arn
}

output "scheduler_role_arn" {
  description = "ARN of the EventBridge scheduler role"
  value       = aws_iam_role.scheduler.arn
}

output "log_group_name" {
  description = "CloudWatch log group name"
  value       = aws_cloudwatch_log_group.pipeline.name
}

output "schedule_arn" {
  description = "ARN of the EventBridge schedule"
  value       = aws_scheduler_schedule.pipeline.arn
}

output "security_group_id" {
  description = "Security group ID for the ECS task"
  value       = aws_security_group.pipeline.id
}
