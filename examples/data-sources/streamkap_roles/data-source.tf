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

# List every role available to the tenant.
data "streamkap_roles" "all" {}

# Index by key so a client credential can name the role it wants rather than
# depending on the order the API returns roles in.
locals {
  role_ids_by_key = { for role in data.streamkap_roles.all.roles : role.key => role.id }
}

resource "streamkap_client_credential" "exporter" {
  role_ids    = [local.role_ids_by_key["ReadOnly"]]
  description = "Read-only token for the metrics exporter"
}

output "role_keys" {
  value = [for role in data.streamkap_roles.all.roles : role.key]
}
