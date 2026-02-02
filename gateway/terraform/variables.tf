variable "aws_region" {
  type        = string
  description = "AWS region"
}

variable "project_name" {
  type        = string
  description = "Project prefix for resources"
  default     = "back-sorte"
}
