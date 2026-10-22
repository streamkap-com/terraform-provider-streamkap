# Two of Google Ads' sign-ins: a refresh token you minted with your own OAuth
# client, and Connect with Google through that client (auth_mode "oauth").

terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
      # Set version to the release matching this documentation.
      # Beta releases require an exact prerelease version; omission selects stable.
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

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

# Connect with Google through your own OAuth client (auth_mode "oauth"): get the grant with the Streamkap CLI right before applying (the
# "Get an OAuth grant" guide has the steps); Terraform cannot run the consent page.
variable "google_ads_oauth_grant_id" {
  type      = string
  sensitive = true
}

resource "streamkap_source_google_ads" "oauth" {
  name           = "google-ads-oauth"
  auth_mode      = "oauth"
  client_id      = "1234567890-abc.apps.googleusercontent.com"
  client_secret  = var.google_ads_client_secret
  oauth_grant_id = var.google_ads_oauth_grant_id
  customer_id    = "1234567890"
  resources      = ["campaign", "campaign_performance"]
}

output "google_ads_source_id" {
  value = streamkap_source_google_ads.example.id
}
