# Minimal Cloudflare R2 destination configuration

resource "streamkap_destination_r2" "example" {
  name                  = "my-r2-dest"
  r2_account            = var.r2_account
  aws_access_key_id     = var.r2_access_key_id
  aws_secret_access_key = var.r2_secret_access_key
  aws_s3_bucket_name    = var.r2_bucket_name
}

variable "r2_account" {
  description = "Cloudflare account ID"
  type        = string
}

variable "r2_access_key_id" {
  description = "R2 access key ID"
  type        = string
}

variable "r2_secret_access_key" {
  description = "R2 secret access key"
  type        = string
  sensitive   = true
}

variable "r2_bucket_name" {
  description = "R2 bucket name"
  type        = string
}
