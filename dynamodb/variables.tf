variable "aws_region" {
  type        = string
  description = "AWS region to create the DynamoDB table."
  default     = "us-east-1"
}

variable "table_name" {
  type        = string
  description = "DynamoDB table name."
  default     = "core"
}

variable "s3_bucket_name" {
  type        = string
  description = "S3 bucket name for DynamoDB exports."
  default     = "bd-thepuregrace-v1-dinamodb-core"
}

variable "export_prefix_base" {
  type        = string
  description = "Base prefix for exports inside the bucket."
  default     = "exports/core"
}

variable "export_retention_days" {
  type        = number
  description = "Days to keep exported objects in S3."
  default     = 30
}

variable "export_format" {
  type        = string
  description = "DynamoDB export format."
  default     = "AMAZON_ION"
}

variable "schedule_expression" {
  type        = string
  description = "EventBridge schedule expression in UTC."
  default     = "cron(10 6 * * ? *)"
}

variable "enable_pitr" {
  type        = bool
  description = "Enable DynamoDB point-in-time recovery (required for ExportTableToPointInTime)."
  default     = true
}

variable "tags" {
  type        = map(string)
  description = "Tags applied to the DynamoDB table."
  default     = {}
}
