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

variable "stripe_restricted_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_stripe" "example" {
  name           = "stripe-billing"
  token          = var.stripe_restricted_key
  resources      = ["customers", "charges", "subscriptions", "invoices"]
  api_version    = "2025-08-27.basil"
  backfill_start = "2026-01-01"
}

output "stripe_source_id" {
  value = streamkap_source_stripe.example.id
}
