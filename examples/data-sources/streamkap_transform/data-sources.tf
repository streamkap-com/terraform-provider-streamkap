terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      # Set version to the release matching this documentation.
      # Beta releases require an exact prerelease version; omission selects stable.
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

variable "transform_id" {
  type        = string
  description = "ID of an existing Streamkap transform, from the UI or API."
}

data "streamkap_transform" "example-transform" {
  id = var.transform_id
}

output "example-transform" {
  value = data.streamkap_transform.example-transform
}