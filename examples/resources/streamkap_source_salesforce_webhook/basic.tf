# Salesforce webhook source. The OAuth fields are technically Optional in the
# generated schema (the backend ships them as `required: true` + `default: ""`,
# which the Terraform Plugin Framework requires us to demote to Optional+
# Computed+Default("") since Required attributes can't carry defaults). Leaving
# them blank produces an unusable connector at runtime, so set them here.
resource "streamkap_source_salesforce_webhook" "example" {
  name = "my-salesforce-webhook"

  camel_source_snapshot_salesforce_instance_url     = "https://your-org.my.salesforce.com"
  camel_source_snapshot_salesforce_auth_client_id   = var.salesforce_client_id
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
