variable "netsuite_private_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_netsuite" "example" {
  name           = "netsuite-customers"
  account_id     = "1234567"
  auth_mode      = "certificate"
  client_id      = "integration-client-id"
  certificate_id = "certificate-id"
  private_key    = var.netsuite_private_key
  resources      = ["customer"]
}
