# Complete Stripe webhook source configuration.
# Shows the commonly used options for ingesting Stripe webhook events.

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

variable "stripe_api_key" {
  type        = string
  sensitive   = true
  description = "Stripe secret API key used to snapshot existing data."
}

variable "stripe_signing_secret" {
  type        = string
  sensitive   = true
  description = "Stripe webhook signing secret used to verify payloads."
}

resource "streamkap_source_stripe_webhook" "example-source-stripe-webhook" {
  name = "example-source-stripe-webhook"

  # Authentication
  camel_source_snapshot_stripe_api_key              = var.stripe_api_key
  camel_source_payload_router_stripe_signing_secret = var.stripe_signing_secret

  # Topic routing
  topic_include_list                                     = "customer,payment_intent,charge,invoice"
  camel_source_payload_router_unknown_type_behavior      = "DEFAULT_TOPIC"
  camel_source_payload_router_unknown_type_default_topic = "unknown"
  camel_source_payload_router_include_event              = true

  # Dead-letter queue for unprocessable events
  camel_source_dlq_enabled = true
}

output "example-source-stripe-webhook" {
  value = streamkap_source_stripe_webhook.example-source-stripe-webhook.id
}
