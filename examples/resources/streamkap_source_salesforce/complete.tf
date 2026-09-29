variable "salesforce_domain" {
  type = string
}

variable "salesforce_client_id" {
  type = string
}

variable "salesforce_client_secret" {
  type      = string
  sensitive = true
}

resource "streamkap_source_salesforce" "example" {
  name           = "salesforce-crm"
  auth_mode      = "service"
  domain         = var.salesforce_domain
  client_id      = var.salesforce_client_id
  client_secret  = var.salesforce_client_secret
  resources      = ["Account", "Contact", "Custom_Record__c"]
  backfill_start = "2026-01-01T00:00:00Z"
}

output "salesforce_source_id" {
  value = streamkap_source_salesforce.example.id
}
