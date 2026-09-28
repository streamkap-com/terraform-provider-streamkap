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
