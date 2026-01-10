# Complete Webhook source configuration
# This example shows all available configuration options for receiving data
# via webhook endpoints

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

resource "streamkap_source_webhook" "example-source-webhook" {
  # Display name for this source in Streamkap UI
  name = "example-source-webhook"

  # Note: webhook_url and api_key are generated after creation
  # and will be available as computed attributes

  # Topic names for output
  topic_include_list = "webhook-orders,webhook-events"

  # Data format
  format = "json" # Options: json, string

  # Message header key field
  camel_source_camel_message_header_key = "key"

  # Add delete field when DELETE HTTP method is used
  transforms_infer_schema_add_delete_field = false
}

output "example-source-webhook-id" {
  value = streamkap_source_webhook.example-source-webhook.id
}

# The webhook URL will be available after creation
output "example-source-webhook-url" {
  value       = streamkap_source_webhook.example-source-webhook.webhook_url
  description = "The webhook URL to send data to"
}

# The API key will be available after creation
output "example-source-webhook-api-key" {
  value       = streamkap_source_webhook.example-source-webhook.api_key
  description = "The API key for authentication"
  sensitive   = true
}
