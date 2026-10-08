# Zendesk signs in only through Connect, which Terraform cannot run. Finish it
# in the Streamkap UI, or run
#   streamkap sources start-source-oauth-connect zendesk --environment acme
# open the returned authorize_url, then run
#   streamkap sources poll-source-oauth-grant zendesk --state <state>
# and apply with the returned grant within 10 minutes.
variable "zendesk_oauth_grant_id" {
  type      = string
  sensitive = true
}

resource "streamkap_source_zendesk" "example" {
  name           = "zendesk-support"
  subdomain      = "acme"
  oauth_grant_id = var.zendesk_oauth_grant_id
  resources      = ["tickets", "users", "organizations"]
}
