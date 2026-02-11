resource "aws_s3_bucket" "dynamodb_exports" {
  bucket = var.s3_bucket_name
  tags   = var.tags
}

resource "aws_s3_bucket_versioning" "dynamodb_exports" {
  bucket = aws_s3_bucket.dynamodb_exports.id
  versioning_configuration {
    status = "Suspended"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "dynamodb_exports" {
  bucket = aws_s3_bucket.dynamodb_exports.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "dynamodb_exports" {
  bucket                  = aws_s3_bucket.dynamodb_exports.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "dynamodb_exports" {
  bucket = aws_s3_bucket.dynamodb_exports.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "dynamodb_exports" {
  bucket = aws_s3_bucket.dynamodb_exports.id
  rule {
    id     = "expire-exports"
    status = "Enabled"

    filter {}

    expiration {
      days = var.export_retention_days
    }
  }
}

data "aws_iam_policy_document" "dynamodb_exports_bucket" {
  statement {
    sid     = "AllowDynamoDBExport"
    effect  = "Allow"
    actions = [
      "s3:PutObject",
      "s3:AbortMultipartUpload"
    ]
    resources = [
      "${aws_s3_bucket.dynamodb_exports.arn}/${var.export_prefix_base}/*"
    ]
    principals {
      type        = "Service"
      identifiers = ["dynamodb.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [data.aws_caller_identity.current.account_id]
    }
    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = [aws_dynamodb_table.core.arn]
    }
  }

  statement {
    sid     = "AllowDynamoDBExportBucketRead"
    effect  = "Allow"
    actions = [
      "s3:GetBucketLocation",
      "s3:ListBucket"
    ]
    resources = [
      aws_s3_bucket.dynamodb_exports.arn
    ]
    principals {
      type        = "Service"
      identifiers = ["dynamodb.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "aws:SourceAccount"
      values   = [data.aws_caller_identity.current.account_id]
    }
    condition {
      test     = "ArnLike"
      variable = "aws:SourceArn"
      values   = [aws_dynamodb_table.core.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "dynamodb_exports" {
  bucket = aws_s3_bucket.dynamodb_exports.id
  policy = data.aws_iam_policy_document.dynamodb_exports_bucket.json
}
