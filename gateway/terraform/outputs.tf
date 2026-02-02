output "http_api_endpoint" {
  description = "Base URL do API Gateway HTTP"
  value       = aws_apigatewayv2_api.http.api_endpoint
}
