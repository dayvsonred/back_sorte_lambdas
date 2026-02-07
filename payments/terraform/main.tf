provider "aws" {
  region = var.aws_region
}

locals {
  base_url = var.stage_name == "$default" ? data.aws_apigatewayv2_api.http.api_endpoint : "${data.aws_apigatewayv2_api.http.api_endpoint}/${var.stage_name}"
}

data "aws_apigatewayv2_api" "http" {
  api_id = var.api_id
}

data "aws_dynamodb_table" "core" {
  name = var.dynamo_table_name
}

resource "aws_iam_role" "lambda_role" {
  name = "${var.project_name}-payments-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_policy" "lambda_dynamo_policy" {
  name = "${var.project_name}-payments-dynamo"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:TransactWriteItems"
        ]
        Resource = [
          data.aws_dynamodb_table.core.arn,
          "${data.aws_dynamodb_table.core.arn}/*"
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_dynamo" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = aws_iam_policy.lambda_dynamo_policy.arn
}

resource "aws_lambda_function" "payments" {
  function_name = "${var.project_name}-payments"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_zip
  source_code_hash = filebase64sha256(var.lambda_zip)

  environment {
    variables = {
      STRIPE_SECRET_KEY    = var.stripe_secret_key
      DYNAMO_TABLE_NAME    = var.dynamo_table_name
      ENV                  = var.env
    }
  }
}

resource "aws_apigatewayv2_integration" "payments" {
  api_id                 = var.api_id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.payments.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "donations" {
  api_id    = var.api_id
  route_key = "POST /payments/donations"
  target    = "integrations/${aws_apigatewayv2_integration.payments.id}"
}

resource "aws_apigatewayv2_route" "intents" {
  api_id    = var.api_id
  route_key = "POST /payments/intents"
  target    = "integrations/${aws_apigatewayv2_integration.payments.id}"
}

resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowAPIGatewayPayments"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.payments.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${data.aws_apigatewayv2_api.http.execution_arn}/*/*/payments/*"
}

resource "aws_cloudwatch_event_bus" "stripe" {
  name              = var.event_source_name
  event_source_name = var.event_source_name
}

resource "aws_cloudwatch_event_rule" "stripe" {
  name           = "${var.project_name}-stripe-events"
  event_bus_name = aws_cloudwatch_event_bus.stripe.name
  event_pattern = jsonencode({
    "source"      = [var.event_source_name]
    "detail-type" = ["payment_intent.succeeded", "payment_intent.payment_failed"]
  })
}

resource "aws_cloudwatch_event_target" "stripe_lambda" {
  rule           = aws_cloudwatch_event_rule.stripe.name
  event_bus_name = aws_cloudwatch_event_bus.stripe.name
  arn            = aws_lambda_function.payments.arn
}

resource "aws_cloudwatch_log_group" "stripe_events" {
  name              = "/aws/eventbridge/${var.project_name}-stripe-events"
  retention_in_days = 14
}

resource "aws_cloudwatch_log_resource_policy" "eventbridge_to_logs" {
  policy_name = "${var.project_name}-eventbridge-logs"
  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          Service = "events.amazonaws.com"
        }
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "${aws_cloudwatch_log_group.stripe_events.arn}:*"
      }
    ]
  })
}

resource "aws_cloudwatch_event_target" "stripe_logs" {
  rule           = aws_cloudwatch_event_rule.stripe.name
  event_bus_name = aws_cloudwatch_event_bus.stripe.name
  arn            = aws_cloudwatch_log_group.stripe_events.arn
}

resource "aws_lambda_permission" "eventbridge" {
  statement_id  = "AllowEventBridgePayments"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.payments.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.stripe.arn
}
