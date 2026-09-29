# The grant is spent when the source is created. Leave it in the
# configuration afterwards, or remove it: either keeps the connection. Set a
# new grant only to reconnect the source.
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
