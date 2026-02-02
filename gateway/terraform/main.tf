terraform {
  required_version = ">= 1.3.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

# Reusa Lambdas existentes criadas nas pastas individuais

data "aws_lambda_function" "users" {
  function_name = "${var.project_name}-users"
}

data "aws_lambda_function" "login" {
  function_name = "${var.project_name}-login"
}

data "aws_lambda_function" "donation" {
  function_name = "${var.project_name}-donation"
}

data "aws_lambda_function" "pix" {
  function_name = "${var.project_name}-pix"
}

data "aws_lambda_function" "contact" {
  function_name = "${var.project_name}-contact"
}

resource "aws_apigatewayv2_api" "http" {
  name          = "${var.project_name}-gateway-http"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "users" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = data.aws_lambda_function.users.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "login" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = data.aws_lambda_function.login.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "donation" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = data.aws_lambda_function.donation.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "pix" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = data.aws_lambda_function.pix.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "contact" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = data.aws_lambda_function.contact.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "users" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /users/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.users.id}"
}

resource "aws_apigatewayv2_route" "users_base" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /users"
  target    = "integrations/${aws_apigatewayv2_integration.users.id}"
}

resource "aws_apigatewayv2_route" "login" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /login/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.login.id}"
}

resource "aws_apigatewayv2_route" "login_base" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /login"
  target    = "integrations/${aws_apigatewayv2_integration.login.id}"
}

resource "aws_apigatewayv2_route" "donation" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /donation/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.donation.id}"
}

resource "aws_apigatewayv2_route" "donation_base" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /donation"
  target    = "integrations/${aws_apigatewayv2_integration.donation.id}"
}

resource "aws_apigatewayv2_route" "pix" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /pix/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.pix.id}"
}

resource "aws_apigatewayv2_route" "pix_base" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /pix"
  target    = "integrations/${aws_apigatewayv2_integration.pix.id}"
}

resource "aws_apigatewayv2_route" "contact" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /contact/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.contact.id}"
}

resource "aws_apigatewayv2_route" "contact_base" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /contact"
  target    = "integrations/${aws_apigatewayv2_integration.contact.id}"
}
resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true

  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gw_access.arn
    format = jsonencode({
      requestId        = "$context.requestId"
      routeKey         = "$context.routeKey"
      httpMethod       = "$context.httpMethod"
      path             = "$context.path"
      status           = "$context.status"
      responseLength   = "$context.responseLength"
      integrationError = "$context.integrationErrorMessage"
      errorMessage     = "$context.error.message"
      requestTime      = "$context.requestTime"
      sourceIp         = "$context.identity.sourceIp"
      userAgent        = "$context.identity.userAgent"
    })
  }
}

resource "aws_cloudwatch_log_group" "api_gw_access" {
  name              = "/aws/apigateway/${var.project_name}-gateway-http"
  retention_in_days = 7
}

resource "aws_lambda_permission" "users" {
  statement_id_prefix = "AllowAPIGatewayUsers-"
  action        = "lambda:InvokeFunction"
  function_name = data.aws_lambda_function.users.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

resource "aws_lambda_permission" "login" {
  statement_id_prefix = "AllowAPIGatewayLogin-"
  action        = "lambda:InvokeFunction"
  function_name = data.aws_lambda_function.login.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

resource "aws_lambda_permission" "donation" {
  statement_id_prefix = "AllowAPIGatewayDonation-"
  action        = "lambda:InvokeFunction"
  function_name = data.aws_lambda_function.donation.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

resource "aws_lambda_permission" "pix" {
  statement_id_prefix = "AllowAPIGatewayPix-"
  action        = "lambda:InvokeFunction"
  function_name = data.aws_lambda_function.pix.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

resource "aws_lambda_permission" "contact" {
  statement_id_prefix = "AllowAPIGatewayContact-"
  action        = "lambda:InvokeFunction"
  function_name = data.aws_lambda_function.contact.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}
