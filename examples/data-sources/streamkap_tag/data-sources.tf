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

data "streamkap_tag" "example-tag" {
  # id = "670e5ca40afe1d3983ce0c22" # Development tag
  id = "670e5bab0d119c0d1f8cda9d" # Production tag
}

output "example-tag" {
  value = data.streamkap_tag.example-tag
}