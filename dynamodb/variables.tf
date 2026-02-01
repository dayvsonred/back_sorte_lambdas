variable "aws_region" {
  type        = string
  description = "AWS region to create the DynamoDB table."
}

variable "table_name" {
  type        = string
  description = "DynamoDB table name."
  default     = "core"
}

variable "tags" {
  type        = map(string)
  description = "Tags applied to the DynamoDB table."
  default     = {}
}
