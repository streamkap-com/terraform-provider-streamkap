variable "hubspot_private_app_token" {
  type      = string
  sensitive = true
}

resource "streamkap_source_hubspot" "example" {
  name                = "hubspot-crm"
  token               = var.hubspot_private_app_token
  resources           = ["contacts", "companies"]
  sync_all_properties = true
  backfill_start      = "2026-01-01T00:00:00Z"
}

output "hubspot_source_id" {
  value = streamkap_source_hubspot.example.id
}
