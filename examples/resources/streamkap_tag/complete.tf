# Complete tag configuration

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

resource "streamkap_tag" "example" {
  name        = "production"
  description = "Tag for production environment resources"

  # Entity types this tag can be applied to
  # Valid values: sources, destinations, pipelines
  type = ["sources", "destinations", "pipelines"]
}

output "tag_id" {
  value = streamkap_tag.example.id
}
