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

resource "aws_iam_role_policy" "lambda_s3_upload" {
  name = "${var.project_name}-donation-s3-upload"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:PutObjectAcl",
          "s3:AbortMultipartUpload",
          "s3:ListBucketMultipartUploads",
          "s3:ListMultipartUploadParts"
        ]
        Resource = [
          "arn:aws:s3:::${var.aws_bucket_name_img_doacao}",
          "arn:aws:s3:::${var.aws_bucket_name_img_doacao}/doacoes/*"
        ]
      }
    ]
  })
}

resource "aws_sqs_queue" "email_events" {
  name                       = var.email_events_queue_name
  visibility_timeout_seconds = 120
  message_retention_seconds  = 1209600
}

resource "aws_iam_role_policy" "lambda_sqs_publish" {
  name = "${var.project_name}-donation-sqs-publish"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:SendMessage"
        ]
        Resource = aws_sqs_queue.email_events.arn
      }
    ]
  })
}

resource "aws_lambda_function" "donation" {
  function_name    = "${var.project_name}-donation"
  role             = aws_iam_role.lambda_role.arn
  handler          = "bootstrap"
  runtime          = "provided.al2"
  filename         = var.lambda_zip
  source_code_hash = filebase64sha256(var.lambda_zip)

  environment {
    variables = {
      DYNAMODB_TABLE             = var.dynamodb_table
      AWS_BUCKET_NAME_IMG_DOACAO = var.aws_bucket_name_img_doacao
      JWT_SECRET                 = var.jwt_secret
      EMAIL_EVENTS_QUEUE_URL     = aws_sqs_queue.email_events.url
      APP_BASE_URL               = var.app_base_url
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
