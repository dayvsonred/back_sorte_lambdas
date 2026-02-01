variable "aws_region" { type = string }
variable "project_name" { type = string default = "back-sorte" }
variable "dynamodb_table" { type = string default = "core" }
variable "efi_client_id" { type = string default = "" }
variable "efi_client_secret" { type = string default = "" }
variable "efi_sandbox" { type = string default = "false" }
variable "efi_timeout" { type = string default = "30" }
variable "efi_ca_pem" { type = string default = "" }
variable "efi_key_pem" { type = string default = "" }
variable "lambda_zip" { type = string }
