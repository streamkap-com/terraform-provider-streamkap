variable "stripe_restricted_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_stripe" "example" {
  name      = "stripe-billing"
  token     = var.stripe_restricted_key
  resources = ["customers", "charges", "subscriptions"]
}
