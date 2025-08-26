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

variable "s3_aws_access_key" {
  type        = string
  description = "The AWS Access Key ID used to connect to S3"
}
variable "s3_aws_secret_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to S3"
}

resource "streamkap_destination_s3" "example-destination-s3" {
  name           = "example-destination-s3"
  aws_access_key = var.s3_aws_access_key
  aws_secret_key = var.s3_aws_secret_key
  aws_region     = "us-west-2"
  bucket_name    = "bucketname"
  format         = "JSON Array"
  output_fields  = ["value", "key"]
}

output "example-destination-s3" {
  value = streamkap_destination_s3.example-destination-s3.id
}