output "dynamodb_table_name" {
  value       = aws_dynamodb_table.core.name
  description = "DynamoDB table name."
}

output "dynamodb_table_arn" {
  value       = aws_dynamodb_table.core.arn
  description = "DynamoDB table ARN."
}

output "bucket_name" {
  value       = aws_s3_bucket.dynamodb_exports.bucket
  description = "S3 bucket for DynamoDB exports."
}

output "lambda_name" {
  value       = aws_lambda_function.export_dynamodb.function_name
  description = "Lambda function name for DynamoDB exports."
}

output "event_rule_name" {
  value       = aws_cloudwatch_event_rule.export_schedule.name
  description = "EventBridge rule name for daily exports."
}
