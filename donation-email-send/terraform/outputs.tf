output "lambda_name" {
  value = aws_lambda_function.email_send.function_name
}

output "pending_schedule_rule" {
  value = aws_cloudwatch_event_rule.pending_schedule.name
}
