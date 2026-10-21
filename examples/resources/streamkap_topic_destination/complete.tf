variable "salesforce_client_secret" {
  type      = string
  sensitive = true
}

variable "destination_id" {
  type = string
}

resource "streamkap_source_salesforce" "example" {
  name          = "salesforce-accounts"
  auth_mode     = "service"
  domain        = "https://example.my.salesforce.com"
  client_id     = "external-client-app-key"
  client_secret = var.salesforce_client_secret
  resources     = ["Account", "Contact"]
}

# One link per resource of the source. Referencing the source also makes
# Terraform remove the links before it deletes the source.
resource "streamkap_topic_destination" "salesforce" {
  for_each       = toset(streamkap_source_salesforce.example.resources)
  topic_id       = "source_${streamkap_source_salesforce.example.id}.salesforce.${each.value}"
  destination_id = var.destination_id
}
