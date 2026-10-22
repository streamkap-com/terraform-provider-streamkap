# Saving the source spends the grant. Keep it in the configuration or remove
# it; set a new one only to reconnect or to change subdomain.

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

variable "zendesk_oauth_grant_id" {
  type      = string
  sensitive = true
}

resource "streamkap_source_zendesk" "example" {
  name           = "zendesk-support"
  subdomain      = "acme"
  oauth_grant_id = var.zendesk_oauth_grant_id
  resources      = ["tickets", "users", "organizations", "groups", "ticket_fields"]
  backfill_start = "2026-01-01"
}

output "zendesk_source_id" {
  value = streamkap_source_zendesk.example.id
}
