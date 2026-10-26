variable "salesforce_client_secret" {
  type      = string
  sensitive = true
}

resource "streamkap_source_salesforce" "example" {
  name          = "salesforce-accounts"
  auth_mode     = "service"
  domain        = "https://example.my.salesforce.com"
  client_id     = "external-client-app-key"
  client_secret = var.salesforce_client_secret
  resources     = ["Account"]
}
