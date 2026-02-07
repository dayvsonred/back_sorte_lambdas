variable "aws_region" {
  type = string
}

variable "project_name" {
  type    = string
  default = "thepuregrace"
}

variable "api_id" {
  type = string
}

variable "stage_name" {
  type    = string
  default = "$default"
}

variable "lambda_zip" {
  type = string
}

variable "stripe_secret_key" {
  type = string
}

variable "env" {
  type    = string
  default = "dev"
}

variable "dynamo_table_name" {
  type    = string
  default = "core"
}

variable "event_source_name" {
  type        = string
  description = "Nome do Partner Event Source da Stripe (ex: aws.partner/stripe.com/ACCOUNT_ID/...)"
}
