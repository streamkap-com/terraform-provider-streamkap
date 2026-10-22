variable "google_ads_service_account_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_google_ads" "example" {
  name                = "google-ads"
  auth_mode           = "service_account"
  customer_id         = "1234567890"
  service_account_key = var.google_ads_service_account_key
  resources           = ["campaign", "ad_group", "campaign_performance"]
}
