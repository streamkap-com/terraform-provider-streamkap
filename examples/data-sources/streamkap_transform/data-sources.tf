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

data "streamkap_transform" "example-transform" {
  id = "660ab64aeb8783e6b76abee3"
}

output "example-transform" {
  value = data.streamkap_transform.example-transform
}