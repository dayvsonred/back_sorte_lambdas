data "archive_file" "export_lambda" {
  type        = "zip"
  source_file = "${path.module}/lambda/export_dynamodb.py"
  output_path = "${path.module}/lambda/export_dynamodb.zip"
}

resource "aws_iam_role" "export_lambda" {
  name = "dynamodb-export-lambda-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
      Action = "sts:AssumeRole"
    }]
  })
  tags = var.tags
}

resource "aws_iam_policy" "export_lambda" {
  name = "dynamodb-export-lambda-policy"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:DescribeTable",
          "dynamodb:ExportTableToPointInTime"
        ]
        Resource = aws_dynamodb_table.core.arn
      },
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })
  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "export_lambda" {
  role       = aws_iam_role.export_lambda.name
  policy_arn = aws_iam_policy.export_lambda.arn
}

resource "aws_lambda_function" "export_dynamodb" {
  function_name = "dynamodb-export-core"
  role          = aws_iam_role.export_lambda.arn
  handler       = "export_dynamodb.handler"
  runtime       = "python3.12"
  filename      = data.archive_file.export_lambda.output_path
  source_code_hash = data.archive_file.export_lambda.output_base64sha256
  timeout       = 60

  environment {
    variables = {
      TABLE_NAME         = var.table_name
      BUCKET_NAME        = var.s3_bucket_name
      EXPORT_PREFIX_BASE = var.export_prefix_base
      EXPORT_FORMAT      = var.export_format
    }
  }

  tags = var.tags
}

resource "aws_cloudwatch_event_rule" "export_schedule" {
  name                = "dynamodb-export-core-daily"
  schedule_expression = var.schedule_expression
  tags                = var.tags
}

resource "aws_cloudwatch_event_target" "export_schedule" {
  rule      = aws_cloudwatch_event_rule.export_schedule.name
  target_id = "export-dynamodb-core"
  arn       = aws_lambda_function.export_dynamodb.arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.export_dynamodb.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.export_schedule.arn
}
