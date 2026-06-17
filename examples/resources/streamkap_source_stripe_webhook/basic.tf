# Stripe webhook source. All fields are Optional in the generated schema (the
# backend ships them as `required: true` + `default: ""`, which the Terraform
# Plugin Framework requires us to demote to Optional+Computed+Default), but the
# Stripe API key and the webhook signing secret are needed for a working
# connector, so set them here.
resource "streamkap_source_stripe_webhook" "example" {
  name = "my-stripe-webhook"

  camel_source_snapshot_stripe_api_key              = var.stripe_api_key
  camel_source_payload_router_stripe_signing_secret = var.stripe_signing_secret
}

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
