variable "facebook_ads_access_token" {
  type      = string
  sensitive = true
}

resource "streamkap_source_facebook_ads" "example" {
  name         = "meta-ads"
  account_id   = "1234567890"
  access_token = var.facebook_ads_access_token
  resources    = ["campaigns", "adsets", "ads"]
}
