variable "aws_region" {
  type = string
}

variable "bucket_name_images" {
  type    = string
  default = "imgs-docao-post-v1"
}

variable "bucket_name_users_profile" {
  type    = string
  default = "doacao-users-prefil-v1"
}
