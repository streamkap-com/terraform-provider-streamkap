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

variable "tag_id" {
  type        = string
  description = "ID of an existing Streamkap tag, from the UI or API."
}

data "streamkap_tag" "example-tag" {
  id = var.tag_id
}

output "example-tag" {
  value = data.streamkap_tag.example-tag
}