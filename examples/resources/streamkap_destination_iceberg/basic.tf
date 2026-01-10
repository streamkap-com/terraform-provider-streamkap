# Minimal Iceberg destination configuration

resource "streamkap_destination_iceberg" "example" {
  name                      = "my-iceberg-dest"
  iceberg_catalog_warehouse = var.iceberg_warehouse
  table_name_prefix         = "analytics"
}

variable "iceberg_warehouse" {
  description = "S3 bucket path for Iceberg warehouse (e.g., s3://bucket/path)"
  type        = string
}
