terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "iceberg_catalog_s3_access_key_id" {
  type        = string
  description = "The AWS Access Key ID used to connect to S3 for Iceberg catalog"
}
variable "iceberg_catalog_s3_secret_access_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to S3 for Iceberg catalog"
}

# Complete Iceberg destination configuration with all options
resource "streamkap_destination_iceberg" "example" {
  name = "example-destination-iceberg"

  # Catalog configuration
  # Valid catalog types: rest, hive, glue
  iceberg_catalog_type = "rest"
  iceberg_catalog_name = "my_catalog"
  iceberg_catalog_uri  = "https://catalog.example.com"

  # AWS credentials for S3 access (required for rest and hive catalog types)
  iceberg_catalog_s3_access_key_id     = var.iceberg_catalog_s3_access_key_id
  iceberg_catalog_s3_secret_access_key = var.iceberg_catalog_s3_secret_access_key

  # AWS region - valid values: ap-south-1, eu-west-2, eu-west-1, ap-northeast-2,
  #              ap-northeast-1, ca-central-1, sa-east-1, cn-north-1, us-gov-west-1,
  #              ap-southeast-1, ap-southeast-2, eu-central-1, us-east-1, us-east-2,
  #              us-west-1, us-west-2
  iceberg_catalog_client_region = "us-west-2"

  # S3 warehouse path (required)
  iceberg_catalog_warehouse = "s3://my-bucket/iceberg-warehouse"

  # Schema/database prefix for table names (required)
  table_name_prefix = "analytics"

  # Insert mode: insert or upsert
  insert_mode = "upsert"

  # Optional: Default ID columns for upsert when key fields are not in Kafka messages
  iceberg_tables_default_id_columns = "id,created_at"

  # Optional: IAM role for assume role authentication
  # iceberg_catalog_client_assume_role_arn = "arn:aws:iam::123456789012:role/my-role"
}

output "iceberg_destination_id" {
  value = streamkap_destination_iceberg.example.id
}
