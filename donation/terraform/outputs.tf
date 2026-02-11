output "api_endpoint" {
  value = aws_apigatewayv2_api.http.api_endpoint
}

output "email_events_queue_arn" {
  value = aws_sqs_queue.email_events.arn
}

output "email_events_queue_url" {
  value = aws_sqs_queue.email_events.url
}
