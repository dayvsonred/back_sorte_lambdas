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

data "aws_caller_identity" "current" {}

resource "aws_iam_role" "lambda_role" {
  name = "${var.project_name}-donation-email-send-role"

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

resource "aws_iam_role_policy" "lambda_policy" {
  name = "${var.project_name}-donation-email-send-policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
          "sqs:ChangeMessageVisibility"
        ]
        Resource = var.queue_arn
      },
      {
        Effect = "Allow"
        Action = [
          "ses:SendEmail",
          "ses:SendTemplatedEmail",
          "ses:SendRawEmail"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query"
        ]
        Resource = "arn:aws:dynamodb:${var.aws_region}:${data.aws_caller_identity.current.account_id}:table/${var.dynamodb_table}"
      }
    ]
  })
}

resource "aws_lambda_function" "email_send" {
  function_name    = "${var.project_name}-donation-email-send"
  role             = aws_iam_role.lambda_role.arn
  handler          = "bootstrap"
  runtime          = "provided.al2"
  filename         = var.lambda_zip
  source_code_hash = filebase64sha256(var.lambda_zip)
  timeout          = 120

  environment {
    variables = {
      DYNAMODB_TABLE    = var.dynamodb_table
      SES_FROM_EMAIL    = var.ses_from_email
      APP_BASE_URL      = var.app_base_url
      DAILY_EMAIL_LIMIT = tostring(var.daily_email_limit)
    }
  }
}

resource "aws_lambda_event_source_mapping" "sqs_consumer" {
  event_source_arn = var.queue_arn
  function_name    = aws_lambda_function.email_send.arn
  batch_size       = 10
}

resource "aws_cloudwatch_event_rule" "pending_schedule" {
  name                = "${var.project_name}-donation-email-send-pending-7am"
  description         = "Process pending donation emails at 07:00 America/Sao_Paulo (10:00 UTC)."
  schedule_expression = var.pending_schedule_expression
}

resource "aws_cloudwatch_event_target" "pending_schedule_target" {
  rule      = aws_cloudwatch_event_rule.pending_schedule.name
  target_id = "donation-email-send-pending"
  arn       = aws_lambda_function.email_send.arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridgePendingEmail"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.email_send.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.pending_schedule.arn
}
