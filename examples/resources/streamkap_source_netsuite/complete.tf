variable "netsuite_consumer_key" {
  type      = string
  sensitive = true
}

variable "netsuite_consumer_secret" {
  type      = string
  sensitive = true
}

variable "netsuite_token_id" {
  type      = string
  sensitive = true
}

variable "netsuite_token_secret" {
  type      = string
  sensitive = true
}

resource "streamkap_source_netsuite" "example" {
  name            = "netsuite-records"
  account_id      = "1234567"
  auth_mode       = "tba"
  consumer_key    = var.netsuite_consumer_key
  consumer_secret = var.netsuite_consumer_secret
  token_id        = var.netsuite_token_id
  token_secret    = var.netsuite_token_secret
  resources       = ["customer", "salesOrder"]
  backfill_start  = "2026-01-01T00:00:00Z"
}

output "netsuite_source_id" {
  value = streamkap_source_netsuite.example.id
}
