variable "hubspot_token" {
  type      = string
  sensitive = true
}

resource "streamkap_source_hubspot" "example" {
  name      = "hubspot-contacts"
  token     = var.hubspot_token
  resources = ["contacts"]
}
