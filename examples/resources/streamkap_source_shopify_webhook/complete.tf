# Complete Shopify webhook source configuration.
# Shows the commonly used options for ingesting Shopify webhook events.

terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 3.0.0"
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

variable "shopify_client_secret" {
  type        = string
  sensitive   = true
  description = "Shopify app client secret."
}

variable "shopify_access_token" {
  type        = string
  sensitive   = true
  description = "Shopify Admin API access token."
}

variable "shopify_hmac_secret" {
  type        = string
  sensitive   = true
  description = "Shopify webhook HMAC signing secret used to verify payloads."
}

resource "streamkap_source_shopify_webhook" "example-source-shopify-webhook" {
  name = "example-source-shopify-webhook"

  # Store connection / authentication
  camel_source_snapshot_shopify_store_url     = "https://your-store.myshopify.com"
  camel_source_snapshot_shopify_client_id     = "your-app-client-id"
  camel_source_snapshot_shopify_client_secret = var.shopify_client_secret
  camel_source_snapshot_shopify_access_token  = var.shopify_access_token
  camel_source_snapshot_shopify_api_version   = "2024-10"

  # Webhook payload verification
  camel_source_payload_router_shopify_hmac_secret = var.shopify_hmac_secret

  # Topic routing
  topic_include_list                                     = "orders,products,customers"
  camel_source_payload_router_unknown_type_behavior      = "DEFAULT_TOPIC"
  camel_source_payload_router_unknown_type_default_topic = "unknown"
  camel_source_payload_router_include_event              = false

  # Dead-letter queue for unprocessable events
  camel_source_dlq_enabled = true
}

output "example-source-shopify-webhook" {
  value = streamkap_source_shopify_webhook.example-source-shopify-webhook.id
}
