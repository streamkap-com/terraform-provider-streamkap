# Salesforce webhook source. Supply the Connected App credentials for the initial snapshot.
resource "streamkap_source_salesforce_webhook" "example" {
  name = "my-salesforce-webhook"

  camel_source_snapshot_salesforce_instance_url       = "https://your-org.my.salesforce.com"
  camel_source_snapshot_salesforce_auth_client_id     = var.salesforce_client_id
  camel_source_snapshot_salesforce_auth_client_secret = var.salesforce_client_secret
}

variable "salesforce_client_id" {
  type        = string
  description = "Salesforce Connected App consumer key (client_id)."
}

variable "salesforce_client_secret" {
  type        = string
  sensitive   = true
  description = "Salesforce Connected App consumer secret (client_secret)."
}
