terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "destination_httpsink_oauth2_client_secret" {
  type        = string
  sensitive   = true
  description = "OAuth2 client secret for HTTP authentication"
}

# Complete HTTP Sink destination configuration with all options
resource "streamkap_destination_httpsink" "example" {
  name = "example-destination-httpsink"

  # Endpoint settings (required)
  http_url = "https://api.example.com/webhook"

  # Authentication
  http_authorization_type    = "oauth2"          # Valid values: none, static, oauth2. Default: none
  http_headers_authorization = "Bearer token123" # Static authorization header (for static auth)

  # OAuth2 settings (when http_authorization_type = oauth2)
  oauth2_access_token_url = "https://auth.example.com/oauth/token"
  oauth2_client_id        = "my-client-id"
  oauth2_client_secret    = var.destination_httpsink_oauth2_client_secret
  oauth2_scope            = "read write"

  # Headers
  http_headers_content_type = "application/json" # Default: application/json
  http_headers_additional   = "X-Custom-Header:value1,X-Another-Header:value2"

  # Proxy settings
  http_proxy_host = "proxy.example.com"
  http_proxy_port = 8080

  # Batching configuration
  batching_enabled       = true                 # Enable batching. Default: false
  batch_max_size         = 500                  # Max records per batch (1-1,000,000). Default: 500
  batch_buffering_enabled = true                # Buffer until batch full or timeout. Default: false
  batch_max_time_ms      = 10000                # Max wait before flush in ms. Default: 10000
  batch_prefix           = "["                  # Batch prefix. Default: [
  batch_suffix           = "]"                  # Batch suffix. Default: ]
  batch_separator        = ","                  # Record separator. Default: ,

  # Retry and timeout settings
  max_retries      = 3                          # Max retries on error. Default: 1
  retry_backoff_ms = 3000                       # Wait time between retries in ms. Default: 3000
  http_timeout     = 30                         # Response timeout in seconds. Default: 30

  # Data formatting
  decimal_format = "NUMERIC"                    # Valid values: BASE64, NUMERIC. Default: NUMERIC

  # Error handling
  errors_tolerance = "none"                     # Valid values: none, all. Default: none
}

output "example_destination_httpsink" {
  value = streamkap_destination_httpsink.example.id
}
