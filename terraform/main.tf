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
  name = "${var.project_name}-lambda-role"

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

resource "aws_lambda_function" "users" {
  function_name = "${var.project_name}-users"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_users_zip
  source_code_hash = filebase64sha256(var.lambda_users_zip)

  environment {
    variables = {
      AWS_REGION      = var.aws_region
      DYNAMODB_TABLE  = var.dynamodb_table
      AWS_BUCKET_NAME = var.aws_bucket_name
      AWS_BUCKET_NAME_IMG_DOACAO = var.aws_bucket_name_img_doacao
      PASSWORD_RESET_KEY = var.password_reset_key
      JWT_SECRET = var.jwt_secret
    }
  }
}

resource "aws_lambda_function" "login" {
  function_name = "${var.project_name}-login"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_login_zip
  source_code_hash = filebase64sha256(var.lambda_login_zip)

  environment {
    variables = {
      AWS_REGION     = var.aws_region
      DYNAMODB_TABLE = var.dynamodb_table
      JWT_SECRET     = var.jwt_secret
    }
  }
}

resource "aws_lambda_function" "donation" {
  function_name = "${var.project_name}-donation"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_donation_zip
  source_code_hash = filebase64sha256(var.lambda_donation_zip)

  environment {
    variables = {
      AWS_REGION      = var.aws_region
      DYNAMODB_TABLE  = var.dynamodb_table
      AWS_BUCKET_NAME_IMG_DOACAO = var.aws_bucket_name_img_doacao
      JWT_SECRET = var.jwt_secret
    }
  }
}

resource "aws_lambda_function" "pix" {
  function_name = "${var.project_name}-pix"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_pix_zip
  source_code_hash = filebase64sha256(var.lambda_pix_zip)

  environment {
    variables = {
      AWS_REGION     = var.aws_region
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

resource "aws_lambda_function" "contact" {
  function_name = "${var.project_name}-contact"
  role          = aws_iam_role.lambda_role.arn
  handler       = "bootstrap"
  runtime       = "provided.al2"
  filename      = var.lambda_contact_zip
  source_code_hash = filebase64sha256(var.lambda_contact_zip)

  environment {
    variables = {
      AWS_REGION     = var.aws_region
      DYNAMODB_TABLE = var.dynamodb_table
    }
  }
}

resource "aws_apigatewayv2_api" "http" {
  name          = "${var.project_name}-http"
  protocol_type = "HTTP"
}

resource "aws_apigatewayv2_integration" "users" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.users.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "login" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.login.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "donation" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.donation.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "pix" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.pix.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "contact" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.contact.arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "users" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /users/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.users.id}"
}

resource "aws_apigatewayv2_route" "login" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /login/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.login.id}"
}

resource "aws_apigatewayv2_route" "donation" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /donation/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.donation.id}"
}

resource "aws_apigatewayv2_route" "pix" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /pix/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.pix.id}"
}

resource "aws_apigatewayv2_route" "contact" {
  api_id    = aws_apigatewayv2_api.http.id
  route_key = "ANY /contact/{proxy+}"
  target    = "integrations/${aws_apigatewayv2_integration.contact.id}"
}

resource "aws_apigatewayv2_stage" "default" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
}

resource "aws_lambda_permission" "users" {
  statement_id  = "AllowAPIGatewayUsers"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.users.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*/users/*"
}

resource "aws_lambda_permission" "login" {
  statement_id  = "AllowAPIGatewayLogin"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.login.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*/login/*"
}

resource "aws_lambda_permission" "donation" {
  statement_id  = "AllowAPIGatewayDonation"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.donation.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*/donation/*"
}

resource "aws_lambda_permission" "pix" {
  statement_id  = "AllowAPIGatewayPix"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.pix.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*/pix/*"
}

resource "aws_lambda_permission" "contact" {
  statement_id  = "AllowAPIGatewayContact"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.contact.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*/contact/*"
}
