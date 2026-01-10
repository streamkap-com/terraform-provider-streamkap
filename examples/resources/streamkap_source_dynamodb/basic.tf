# Minimal DynamoDB CDC source configuration
# This example captures changes from DynamoDB tables using DynamoDB Streams

resource "streamkap_source_dynamodb" "example" {
  name = "my-dynamodb-source"

  # AWS credentials
  aws_region        = "us-east-1"
  aws_access_key_id = var.aws_access_key_id
  aws_secret_key    = var.aws_secret_key

  # S3 bucket for export snapshots (required)
  s3_export_bucket_name = "my-streamkap-exports"

  # Tables to capture (comma-separated)
  table_include_list = "Orders,Customers"
}

variable "aws_access_key_id" {
  description = "AWS Access Key ID"
  type        = string
  sensitive   = true
}

variable "aws_secret_key" {
  description = "AWS Secret Access Key"
  type        = string
  sensitive   = true
}
