variable "ga4_service_account_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_google_analytics" "example" {
  name                = "ga4-website"
  property_id         = "123456789"
  property_ids        = ["987654321"]
  service_account_key = var.ga4_service_account_key
  resources           = ["website_overview", "devices", "traffic_sources", "custom_landing_pages"]
  custom_reports = jsonencode([{
    name       = "landing_pages"
    dimensions = ["landingPage", "deviceCategory"]
    metrics    = ["sessions", "engagedSessions"]
  }])
  keep_empty_rows = false
  backfill_start  = "2026-01-01"
}

output "ga4_source_id" {
  value = streamkap_source_google_analytics.example.id
}
