variable "aws_region" {
  type = string
}

variable "project_name" {
  type    = string
  default = "back-sorte"
}

variable "dynamodb_table" {
  type    = string
  default = "core"
}

variable "aws_bucket_name" {
  type    = string
  default = ""
}

variable "aws_bucket_name_img_doacao" {
  type    = string
  default = ""
}

variable "jwt_secret" {
  type    = string
  default = ""
}

variable "password_reset_key" {
  type    = string
  default = ""
}

variable "lambda_zip" {
  type = string
}

variable "email_events_queue_url" {
  type    = string
  default = ""
}

variable "email_events_queue_arn" {
  type    = string
  default = ""
}
