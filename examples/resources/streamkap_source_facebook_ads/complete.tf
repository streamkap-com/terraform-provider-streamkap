variable "facebook_ads_access_token" {
  type      = string
  sensitive = true
}

variable "facebook_app_secret" {
  type      = string
  sensitive = true
}

resource "streamkap_source_facebook_ads" "example" {
  name         = "meta-ads"
  account_id   = "1234567890"
  account_ids  = ["2345678901"]
  access_token = var.facebook_ads_access_token
  # Set app_id and app_secret together, or leave both out.
  app_id                 = "1122334455"
  app_secret             = var.facebook_app_secret
  resources              = ["campaigns", "adsets", "ads", "ad_account", "campaign_insights", "custom_by_country"]
  custom_insights        = jsonencode([{ name = "by_country", level = "campaign", breakdowns = ["country"], fields = ["campaign_id", "spend", "impressions"] }])
  include_deleted        = true
  fetch_thumbnail_images = false
  api_version            = "v26.0"
  backfill_start         = "2026-01-01"
}

output "facebook_ads_source_id" {
  value = streamkap_source_facebook_ads.example.id
}
