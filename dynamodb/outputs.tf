output "dynamodb_table_name" {
  value       = aws_dynamodb_table.core.name
  description = "DynamoDB table name."
}

output "dynamodb_table_arn" {
  value       = aws_dynamodb_table.core.arn
  description = "DynamoDB table ARN."
}
