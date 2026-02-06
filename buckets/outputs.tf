output "bucket_name" {
  value = aws_s3_bucket.images.bucket
}

output "bucket_arn" {
  value = aws_s3_bucket.images.arn
}

output "users_profile_bucket_name" {
  value = aws_s3_bucket.users_profile.bucket
}

output "users_profile_bucket_arn" {
  value = aws_s3_bucket.users_profile.arn
}
