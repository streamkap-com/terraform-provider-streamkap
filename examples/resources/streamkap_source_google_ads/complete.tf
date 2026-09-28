# auth_mode "oauth" needs an oauth_grant_id from Connect with Google
# (Streamkap UI, or the CLI's start-source-oauth-connect and
# poll-source-oauth-grant) on top of your OAuth client. The refresh_token mode
# below takes a refresh token you minted with that client instead.
variable "google_ads_client_secret" {
  type      = string
  sensitive = true
}

variable "google_ads_refresh_token" {
  type      = string
  sensitive = true
}

resource "streamkap_source_google_ads" "example" {
  name                    = "google-ads"
  auth_mode               = "refresh_token"
  client_id               = "1234567890-abc.apps.googleusercontent.com"
  client_secret           = var.google_ads_client_secret
  refresh_token           = var.google_ads_refresh_token
  customer_id             = "1234567890"
  customer_ids            = ["2345678901"]
  login_customer_id       = "3456789012"
  include_client_accounts = false
  resources               = ["campaign", "ad_group", "ad_group_ad", "campaign_performance", "custom_campaign_devices"]
  custom_queries = jsonencode([{
    name  = "campaign_devices"
    query = "SELECT campaign.id, segments.device, metrics.clicks FROM campaign"
  }])
  conversion_window_days = 30
  backfill_start         = "2026-01-01"
}

output "google_ads_source_id" {
  value = streamkap_source_google_ads.example.id
}
