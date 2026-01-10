# Minimal Starburst destination configuration

resource "streamkap_destination_starburst" "example" {
  name                  = "my-starburst-dest"
  aws_access_key_id     = var.aws_access_key_id
  aws_secret_access_key = var.aws_secret_access_key
  aws_s3_region         = var.aws_s3_region
  aws_s3_bucket_name    = var.aws_s3_bucket_name
}

variable "aws_access_key_id" {
  description = "AWS access key ID"
  type        = string
}

variable "aws_secret_access_key" {
  description = "AWS secret access key"
  type        = string
  sensitive   = true
}

variable "aws_s3_region" {
  description = "AWS S3 region"
  type        = string
}

variable "aws_s3_bucket_name" {
  description = "AWS S3 bucket name"
  type        = string
}
