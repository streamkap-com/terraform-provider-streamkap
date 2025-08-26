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

variable "iceberg_aws_access_key" {
  type        = string
  description = "The AWS Access Key ID used to connect to Iceberg. Required for rest and hive."
}
variable "iceberg_aws_secret_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to Iceberg. Required for rest and hive."
}

resource "streamkap_destination_iceberg" "example-destination-iceberg" {
  name           = "example-destination-iceberg"
  catalog_type   = "rest"
  catalog_name   = "iceberg_catalog_name"
  catalog_uri    = "iceberg_catalog_uri"
  aws_access_key = var.iceberg_aws_access_key
  aws_secret_key = var.iceberg_aws_secret_key
  aws_region     = "us-west-2"
  bucket_path    = "iceberg_bucket_path"
  schema         = "iceberg_schema"
}

output "example-destination-iceberg" {
  value = streamkap_destination_iceberg.example-destination-iceberg.id
}