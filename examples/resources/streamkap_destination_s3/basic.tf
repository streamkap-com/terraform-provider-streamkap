# Minimal S3 destination configuration

resource "streamkap_destination_s3" "example" {
  name               = "my-s3-dest"
  aws_access_key_id  = var.aws_access_key_id
  aws_secret_access_key = var.aws_secret_access_key
  aws_s3_bucket_name = var.s3_bucket_name
}

variable "aws_access_key_id" {
  description = "AWS Access Key ID"
  type        = string
}

variable "aws_secret_access_key" {
  description = "AWS Secret Access Key"
  type        = string
  sensitive   = true
}

variable "s3_bucket_name" {
  description = "S3 bucket name"
  type        = string
}
