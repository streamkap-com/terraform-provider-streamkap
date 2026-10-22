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

variable "salesforce_domain" {
  type = string
}

variable "salesforce_client_id" {
  type = string
}

variable "salesforce_client_secret" {
  type      = string
  sensitive = true
}

resource "streamkap_source_salesforce" "example" {
  name           = "salesforce-crm"
  auth_mode      = "service"
  domain         = var.salesforce_domain
  client_id      = var.salesforce_client_id
  client_secret  = var.salesforce_client_secret
  resources      = ["Account", "Contact", "Custom_Record__c"]
  backfill_start = "2026-01-01T00:00:00Z"
}

# JWT bearer: the External Client App's certificate flow, signing in as one user.
variable "salesforce_jwt_private_key" {
  type      = string
  sensitive = true
}

resource "streamkap_source_salesforce" "jwt" {
  name        = "salesforce-crm-jwt"
  auth_mode   = "jwt"
  domain      = var.salesforce_domain
  client_id   = var.salesforce_client_id
  username    = "integration@example.com"
  private_key = var.salesforce_jwt_private_key
  resources   = ["Account", "Opportunity"]
}

# Connect (OAuth): get the grant with the Streamkap CLI right before applying (the
# "Get an OAuth grant" guide has the steps); Terraform cannot run the consent page.
variable "salesforce_oauth_grant_id" {
  type      = string
  sensitive = true
}

resource "streamkap_source_salesforce" "oauth" {
  name           = "salesforce-crm-oauth"
  auth_mode      = "oauth"
  oauth_grant_id = var.salesforce_oauth_grant_id
  resources      = ["Account", "Contact"]
}

output "salesforce_source_id" {
  value = streamkap_source_salesforce.example.id
}
