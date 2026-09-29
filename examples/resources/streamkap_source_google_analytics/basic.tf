variable "ga4_service_account_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_google_analytics" "example" {
  name                = "ga4-website"
  property_id         = "123456789"
  service_account_key = var.ga4_service_account_key
  resources           = ["website_overview", "devices"]
}
