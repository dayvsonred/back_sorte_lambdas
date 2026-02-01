variable "aws_region" { type = string }
variable "project_name" { type = string default = "back-sorte" }
variable "dynamodb_table" { type = string default = "core" }
variable "lambda_zip" { type = string }
