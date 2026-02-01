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
  name = "${var.project_name}-donation-role"

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

resource "aws_iam_role_policy_attachment" "lambda_s3" {
  role       = aws_iam_role.lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
}

resource "aws_lambda_function" "donation" {
  function_name = "${var.project_name}-donation"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_zip
  source_code_hash = filebase64sha256(var.lambda_zip)

  environment {
    variables = {
      AWS_REGION      = var.aws_region
      DYNAMODB_TABLE  = var.dynamodb_table
      AWS_BUCKET_NAME_IMG_DOACAO = var.aws_bucket_name_img_doacao
      JWT_SECRET = var.jwt_secret
    }
  }
}

resource "aws_apigatewayv2_api" "http" {
  name          = "${var.project_name}-donation-http"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "donation" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.donation.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "donation" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /donation/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.donation.id}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_lambda_permission" "donation" {
  statement_id  = "AllowAPIGatewayDonation"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.donation.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*/donation/*"
}
