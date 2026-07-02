# Shopify webhook source. All fields are Optional in the generated schema (the
# backend ships them as `required: true` + `default: ""`, which the Terraform
# Plugin Framework requires us to demote to Optional+Computed+Default), but the
# store URL, an access credential, and the HMAC signing secret are needed for a
# working connector, so set them here.
resource "streamkap_source_shopify_webhook" "example" {
  name = "my-shopify-webhook"

  camel_source_snapshot_shopify_store_url         = "https://your-store.myshopify.com"
  camel_source_snapshot_shopify_access_token      = var.shopify_access_token
  camel_source_payload_router_shopify_hmac_secret = var.shopify_hmac_secret
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
