variable "aws_region" {
  type        = string
  description = "AWS region"
}

variable "project_name" {
  type        = string
  description = "Project prefix for resources"
  default     = "back-sorte"
}

variable "dynamodb_table" {
  type        = string
  description = "DynamoDB table name"
  default     = "core"
}

variable "aws_bucket_name" {
  type        = string
  description = "S3 bucket for profile images"
  default     = ""
}

variable "aws_bucket_name_img_doacao" {
  type        = string
  description = "S3 bucket for donation images"
  default     = ""
}

variable "jwt_secret" {
  type        = string
  description = "JWT secret"
  default     = ""
}

variable "password_reset_key" {
  type        = string
  description = "Password reset key"
  default     = ""
}

variable "efi_client_id" {
  type        = string
  description = "EFI client id"
  default     = ""
}

variable "efi_client_secret" {
  type        = string
  description = "EFI client secret"
  default     = ""
}

variable "efi_sandbox" {
  type        = string
  description = "EFI sandbox flag"
  default     = "false"
}

variable "efi_timeout" {
  type        = string
  description = "EFI timeout"
  default     = "30"
}

variable "efi_ca_pem" {
  type        = string
  description = "EFI CA pem"
  default     = ""
}

variable "efi_key_pem" {
  type        = string
  description = "EFI key pem"
  default     = ""
}

variable "lambda_users_zip" {
  type        = string
  description = "Path to users lambda zip"
}

variable "lambda_login_zip" {
  type        = string
  description = "Path to login lambda zip"
}

variable "lambda_donation_zip" {
  type        = string
  description = "Path to donation lambda zip"
}

variable "lambda_pix_zip" {
  type        = string
  description = "Path to pix lambda zip"
}

variable "lambda_contact_zip" {
  type        = string
  description = "Path to contact lambda zip"
}
