variable "aws_region" {
  type = string
}

variable "project_name" {
  type    = string
  default = "back-sorte"
}

variable "lambda_zip" {
  type = string
}

variable "queue_arn" {
  type = string
}

variable "dynamodb_table" {
  type    = string
  default = "core"
}

variable "ses_from_email" {
  type = string
}

variable "app_base_url" {
  type    = string
  default = "https://www.thepuregrace.com"
}

variable "daily_email_limit" {
  type    = number
  default = 199
}

variable "email_provider" {
  type    = string
  default = "ses"
}

variable "brevo_api_key" {
  type      = string
  default   = ""
  sensitive = true
}

variable "email_from_name" {
  type    = string
  default = "The Pure Grace"
}

variable "pending_schedule_expression" {
  type    = string
  default = "cron(0 10 * * ? *)"
}
