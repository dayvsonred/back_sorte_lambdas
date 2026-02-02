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

resource "aws_iam_role" "lambda_role" {
  name = "${var.project_name}-pix-role"

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

resource "aws_iam_role_policy_attachment" "lambda_dynamo" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"
}

resource "aws_lambda_function" "pix" {
  function_name = "${var.project_name}-pix"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_zip
  source_code_hash = filebase64sha256(var.lambda_zip)

  environment {
    variables = {
      DYNAMODB_TABLE = var.dynamodb_table
      CLIENT_ID      = var.efi_client_id
      CLIENT_SECRET  = var.efi_client_secret
      SANDBOX        = var.efi_sandbox
      TIMEOUT        = var.efi_timeout
      CA_PEM         = var.efi_ca_pem
      KEY_PEM        = var.efi_key_pem
    }
  }
}

resource "aws_apigatewayv2_api" "http" {
  name          = "${var.project_name}-pix-http"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "pix" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.pix.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "pix" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /pix/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.pix.id}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_lambda_permission" "pix" {
  statement_id  = "AllowAPIGatewayPix"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.pix.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*/pix/*"
}
